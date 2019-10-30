// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package plcconnector implements communication with PLC.
package plcconnector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"
)

// PLC .
type PLC struct {
	callback  func(service int, statut int, tag *Tag)
	closeI    bool
	closeMut  sync.RWMutex
	closeWMut sync.Mutex
	closeWait *sync.Cond
	port      uint16
	tMut      sync.RWMutex
	tags      map[string]*Tag

	Class       map[int]Class
	DumpNetwork bool // enables dumping network packets
	Verbose     bool // enables debugging output
	Timeout     time.Duration
}

// Init initialize library. Must be called first.
func Init(eds string, testTags bool) (*PLC, error) {
	var p PLC
	p.Class = make(map[int]Class)
	p.tags = make(map[string]*Tag)
	p.Timeout = 60 * time.Second

	if eds == "" {
		p.Class[1] = defaultIdentityClass()
	} else {
		err := p.loadEDS(eds)
		if err != nil {
			return nil, err
		}
	}

	if testTags {
		p.tags["testBOOL"] = &Tag{Name: "testBOOL", Typ: TypeBOOL, Count: 4, data: []uint8{
			0x00, 0x01, 0xFF, 0x55}}

		p.tags["testSINT"] = &Tag{Name: "testSINT", Typ: TypeSINT, Count: 4, data: []uint8{
			0xFF, 0xFE, 0x00, 0x01}}

		p.tags["testINT"] = &Tag{Name: "testINT", Typ: TypeINT, Count: 10, data: []uint8{
			0xFF, 0xFF, 0x00, 0x01, 0xFE, 0x00, 0xFC, 0x00, 0xCA, 0x00, 0xBD, 0x00, 0xB1, 0x00, 0xFF, 0x00, 127, 0x00, 128, 0x00}}

		p.tags["testDINT"] = &Tag{Name: "testDINT", Typ: TypeDINT, Count: 2, data: []uint8{
			0xFF, 0xFF, 0xFF, 0xFF,
			0x01, 0x00, 0x00, 0x00}}

		p.tags["testREAL"] = &Tag{Name: "testREAL", Typ: TypeREAL, Count: 2, data: []uint8{
			0xa4, 0x70, 0x9d, 0x3f,
			0xcd, 0xcc, 0x44, 0xc1}}

		p.tags["testDWORD"] = &Tag{Name: "testDWORD", Typ: TypeDWORD, Count: 2, data: []uint8{
			0xFF, 0xFF, 0xFF, 0xFF,
			0x01, 0x00, 0x00, 0x00}}

		p.tags["testLINT"] = &Tag{Name: "testLINT", Typ: TypeLINT, Count: 2, data: []uint8{
			0xFF, 0xFF, 0xFF, 0xFF,
			0xFF, 0xFF, 0xFF, 0xFF,
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00}}

		p.tags["testASCII"] = &Tag{Name: "testASCII", Typ: TypeSINT, Count: 17, data: []uint8{
			'H', 'e', 'l', 'l',
			'o', '!', 0x00, 0x01, 0x7F, 0xFE, 0xFC, 0xCA, 0xBD, 0xB1, 0xFF, 127, 128}}
	}

	return &p, nil
}

func (p *PLC) debug(args ...interface{}) {
	if p.Verbose {
		fmt.Println(args...)
	}
}

func (p *PLC) readTag(tag string, count uint16) ([]uint8, uint16, bool) {
	p.tMut.RLock()
	tg, ok := p.tags[tag]
	var (
		tgtyp  uint16
		tgdata []uint8
	)
	if ok {
		p.debug(tag+":", tg)
		tgtyp = uint16(tg.Typ)
		tgdata = make([]uint8, count*typeLen(tgtyp))
		if count > uint16(tg.Count) {
			ok = false
		} else {
			copy(tgdata, tg.data)
		}
	}
	p.tMut.RUnlock()
	if ok {
		if p.callback != nil {
			go p.callback(ReadTag, Success, &Tag{Name: tag, Typ: int(tgtyp), Count: int(count), data: tgdata})
		}
		return tgdata, tgtyp, true
	}
	if p.callback != nil {
		go p.callback(ReadTag, PathSegmentError, nil)
	}
	return nil, 0, false
}

func (p *PLC) saveTag(tag string, typ, count uint16, data []uint8) bool {
	p.tMut.Lock()
	tg, ok := p.tags[tag]
	if ok && tg.Typ == int(typ) && tg.Count >= int(count) {
		copy(tg.data, data)
	} else {
		p.tags[tag] = &Tag{Name: tag, Typ: int(typ), Count: int(count), data: data}
	}
	p.tMut.Unlock()
	if p.callback != nil {
		go p.callback(WriteTag, Success, &Tag{Name: tag, Typ: int(typ), Count: int(count), data: data})
	}
	return true
}

// AddTag adds tag.
func (p *PLC) AddTag(t Tag) {
	size := typeLen(uint16(t.Typ)) * uint16(t.Count)
	t.data = make([]uint8, size)
	p.tMut.Lock()
	p.tags[t.Name] = &t
	p.tMut.Unlock()
}

// UpdateTag sets data to the tag
func (p *PLC) UpdateTag(name string, offset int, data []uint8) bool {
	p.tMut.Lock()
	defer p.tMut.Unlock()
	t, ok := p.tags[name]
	if !ok {
		fmt.Println("plcconnector UpdateTag: no tag named ", name)
		return false
	}
	offset *= int(typeLen(uint16(t.Typ)))
	to := offset + len(data)
	if to > len(t.data) {
		fmt.Println("plcconnector UpdateTag: to large data ", name)
		return false
	}
	for i := offset; i < to; i++ {
		t.data[i] = data[i-offset]
	}
	return true
}

// Callback registers function called at receiving communication with PLC.
// tag may be nil in event of error or reset.
func (p *PLC) Callback(function func(service int, status int, tag *Tag)) {
	p.callback = function
}

// Serve listens on the TCP network address host.
func (p *PLC) Serve(host string) error {
	rand.Seed(time.Now().UnixNano())

	p.closeMut.Lock()
	p.closeI = false
	p.closeMut.Unlock()

	p.closeWait = sync.NewCond(&p.closeWMut)

	sock := net.ListenConfig{}
	sock.Control = func(network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			if err != nil {
				fmt.Println("plcconnector Serve: ", err)
				return
			}
		})
	}
	serv2, err := sock.Listen(context.Background(), "tcp", host)
	if err != nil {
		fmt.Println("plcconnector Serve: ", err)
		return err
	}
	p.port = getPort(host)
	serv := serv2.(*net.TCPListener)
	for {
		serv.SetDeadline(time.Now().Add(time.Second))
		conn, err := serv.AcceptTCP()
		if e, ok := err.(net.Error); ok && e.Timeout() {
			p.closeMut.RLock()
			endP := p.closeI
			p.closeMut.RUnlock()
			if endP {
				break
			}
		} else if err != nil {
			fmt.Println("plcconnector Serve: ", err)
			return err
		} else {
			go p.handleRequest(conn)
		}
	}
	serv.Close()
	p.debug("Serve shutdown")
	p.closeWait.Signal()
	return nil
}

// Close shutdowns server
func (p *PLC) Close() {
	p.closeMut.Lock()
	p.closeI = true
	p.closeMut.Unlock()
	p.closeWait.L.Lock()
	p.closeWait.Wait()
	p.closeWait.L.Unlock()
}

type req struct {
	c        net.Conn
	connID   uint32
	rrdata   sendData
	encHead  encapsulationHeader
	p        *PLC
	readBuf  *bufio.Reader
	writeBuf *bytes.Buffer
}

func (r *req) read(data interface{}) error {
	err := binary.Read(r.readBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	if r.p.DumpNetwork {
		fmt.Printf("%#v\n", data)
	}
	return err
}

func (r *req) write(data interface{}) {
	err := binary.Write(r.writeBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func (r *req) reset() {
	r.readBuf.Reset(r.c)
	r.writeBuf.Reset()
}

func (p *PLC) handleRequest(conn net.Conn) {
	r := req{}
	r.connID = uint32(0)
	r.c = conn
	r.p = p
	r.readBuf = bufio.NewReader(conn)
	r.writeBuf = new(bytes.Buffer)

loop:
	for {
		r.reset()

		p.closeMut.RLock()
		endP := p.closeI
		p.closeMut.RUnlock()
		if endP {
			break loop
		}

		err := conn.SetReadDeadline(time.Now().Add(p.Timeout))
		if err != nil {
			fmt.Println(err)
			break loop
		}

		p.debug()
		err = r.read(&r.encHead)
		if err != nil {
			break loop
		}

	command:
		switch r.encHead.Command {
		case nop:
			if r.eipNOP() != nil {
				break loop
			}
			continue loop

		case registerSession:
			if r.eipRegisterSession() != nil {
				break loop
			}

		case unregisterSession:
			p.debug("UnregisterSession")
			break loop

		case listIdentity: // UDP!
			if r.eipListIdentity() != nil {
				break loop
			}

		case listServices:
			if r.eipListServices() != nil {
				break loop
			}

		case listInterfaces:
			p.debug("ListInterfaces")

			itemCount := uint16(0)
			r.write(itemCount)

		case sendRRData, sendUnitData:
			p.debug("SendRRData/SendUnitData")

			var (
				item         itemType
				dataLen      uint16
				addrLen      uint16
				protd        protocolData
				protSeqCount uint16
				resp         response
			)
			err = r.read(&r.rrdata)
			if err != nil {
				break loop
			}

			r.rrdata.Timeout = 0
			cidok := false
			itemserror := false

			if r.rrdata.ItemCount != 2 {
				p.debug("itemCount != 2")
				r.encHead.Status = 0x03 // Incorrect data
				break command
			}

			// address item
			err = r.read(&item)
			if err != nil {
				break loop
			}
			if item.Type == connAddressItem { // TODO itemdata to connID
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != nullAddressItem {
				p.debug("unkown address item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			// data item
			err = r.read(&item)
			if err != nil {
				break loop
			}
			if item.Type == connDataItem {
				err = r.read(&protSeqCount)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != unconnDataItem {
				p.debug("unkown data item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			if itemserror {
				r.encHead.Status = 0x03 // Incorrect data
				break command
			}

			// CIP
			err = r.read(&protd)
			if err != nil {
				break loop
			}

			protdPath := make([]uint8, protd.PathSize*2)
			err = r.read(&protdPath)
			if err != nil {
				break loop
			}

			resp.Service = protd.Service + 128
			resp.Status = Success

			switch protd.Service {
			case GetAttrAll:
				p.debug("GetAttributesAll")
				var (
					iok bool
					in  *Instance
				)
				c, cok := p.Class[int(protdPath[1])]
				if cok {
					in, iok = c.Inst[int(protdPath[3])]
				}

				if cok && iok {
					p.debug(c.Name, protdPath[3])

					attrdata, attrlen := in.getAttrAll()

					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + attrlen)})
					r.write(resp)
					r.write(attrdata)
				} else {
					p.debug("path unknown", protdPath)

					resp.Status = 0x05 // Path destination unknown

					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
					r.write(resp)
				}

			case GetAttr:
				p.debug("GetAttributesSingle")

				var (
					iok bool
					in  *Instance
					aok bool
					at  *Attribute
				)
				c, cok := p.Class[int(protdPath[1])]
				if cok {
					in, iok = c.Inst[int(protdPath[3])]
					if iok && int(protdPath[5]) < len(in.Attr) {
						at = in.Attr[protdPath[5]]
						aok = true
					}
				}
				resp.Service = protd.Service + 128

				if cok && iok && aok {
					p.debug(c.Name, protdPath[3], at.Name)
					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + len(at.data))})
					r.write(resp)
					r.write(at.data)
				} else {
					p.debug("path unknown", protdPath)

					resp.Status = 0x05 // Path destination unknown

					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
					r.write(resp)
				}

			case InititateUpload: // TODO only File class?
				p.debug("InititateUpload")
				var (
					iok     bool
					in      *Instance
					maxSize uint8
				)
				c, cok := p.Class[int(protdPath[1])]
				if cok {
					in, iok = c.Inst[int(protdPath[3])]
				}

				err = r.read(&maxSize)
				if err != nil {
					break loop
				}

				if cok && iok {
					p.debug(c.Name, protdPath[3], maxSize)

					var sr initUploadResponse
					sr.FileSize = uint32(len(in.data))
					sr.TransferSize = maxSize
					in.argUint8[0] = maxSize // TransferSize
					in.argUint8[1] = 0       // TransferNumber
					in.argUint8[2] = 0       // TransferNumber rollover

					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + binary.Size(sr))})
					r.write(resp)
					r.write(sr)
				} else {
					p.debug("path unknown", protdPath)

					resp.Status = 0x05 // Path destination unknown

					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
					r.write(resp)
				}

			case UploadTransfer: // TODO only File class?
				p.debug("UploadTransfer")
				var (
					iok        bool
					in         *Instance
					transferNo uint8
				)
				c, cok := p.Class[int(protdPath[1])]
				if cok {
					in, iok = c.Inst[int(protdPath[3])]
				}

				err = r.read(&transferNo)
				if err != nil {
					break loop
				}

				if cok && iok {
					if transferNo == in.argUint8[1] || transferNo == in.argUint8[1]+1 || (transferNo == 0 && in.argUint8[1] == 255) {
						p.debug(c.Name, protdPath[3], transferNo)

						if transferNo == 0 && in.argUint8[1] == 255 { // rollover
							p.debug("rollover")
							in.argUint8[2]++ // FIXME retry!
						}

						var sr uploadTransferResponse
						addcksum := false
						dtlen := len(in.data)
						pos := (int(in.argUint8[2]) + 1) * int(transferNo) * int(in.argUint8[0])
						posto := pos + int(in.argUint8[0])
						if posto > dtlen {
							posto = dtlen
						}
						dt := in.data[pos:posto]
						sr.TransferNumber = transferNo
						if transferNo == 0 && dtlen <= int(in.argUint8[0]) {
							sr.TranferPacketType = tptFirstLast
							addcksum = true
						} else if transferNo == 0 && in.argUint8[2] == 0 {
							sr.TranferPacketType = tptFirst
						} else if pos+int(in.argUint8[0]) >= dtlen {
							sr.TranferPacketType = tptLast
							addcksum = true
						} else {
							sr.TranferPacketType = tptMiddle
						}
						in.argUint8[1] = transferNo

						ln := uint16(binary.Size(resp) + binary.Size(sr) + len(dt))
						if addcksum {
							ln += uint16(binary.Size(in.Attr[7].data))
						}

						p.debug(sr)
						p.debug(len(dt), pos, posto)

						r.write(r.rrdata)
						r.write(itemType{Type: nullAddressItem, Length: 0})
						r.write(itemType{Type: unconnDataItem, Length: ln})
						r.write(resp)
						r.write(sr)
						r.write(dt)
						if addcksum {
							r.write(in.Attr[7].data)
						}
					} else {
						p.debug("transfer number error", transferNo)

						resp.Status = 0x20 // Invalid Parameter
						resp.AddStatusSize = 1
						addStatus := uint16(0x06)

						r.write(r.rrdata)
						r.write(itemType{Type: nullAddressItem, Length: 0})
						r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + binary.Size(addStatus))})
						r.write(resp)
						r.write(addStatus)
					}
				} else {
					p.debug("path unknown", protdPath)

					resp.Status = 0x05 // Path destination unknown

					r.write(r.rrdata)
					r.write(itemType{Type: nullAddressItem, Length: 0})
					r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
					r.write(resp)
				}

			case ForwardOpen:
				p.debug("ForwardOpen")

				var (
					fodata forwardOpenData
					sr     forwardOpenResponse
				)
				err = r.read(&fodata)
				if err != nil {
					break loop
				}
				connPath := make([]uint8, fodata.ConnPathSize*2)
				err = r.read(&connPath)
				if err != nil {
					break loop
				}

				sr.OTConnectionID = rand.Uint32()
				sr.TOConnectionID = fodata.TOConnectionID
				sr.ConnSerialNumber = fodata.ConnSerialNumber
				sr.VendorID = fodata.VendorID
				sr.OriginatorSerialNumber = fodata.OriginatorSerialNumber
				sr.OTAPI = fodata.OTRPI
				sr.TOAPI = fodata.TORPI
				sr.AppReplySize = 0

				r.connID = fodata.TOConnectionID

				r.write(r.rrdata)
				r.write(itemType{Type: nullAddressItem, Length: 0})
				r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + binary.Size(sr))})
				r.write(resp)
				r.write(sr)

			case ForwardClose:
				p.debug("ForwardClose")

				var (
					fcdata forwardCloseData
					sr     forwardCloseResponse
				)
				err = r.read(&fcdata)
				if err != nil {
					break loop
				}
				connPath := make([]uint8, fcdata.ConnPathSize*2)
				err = r.read(&connPath)
				if err != nil {
					break loop
				}

				sr.ConnSerialNumber = fcdata.ConnSerialNumber
				sr.VendorID = fcdata.VendorID
				sr.OriginatorSerialNumber = fcdata.OriginatorSerialNumber
				sr.AppReplySize = 0

				r.connID = 0

				r.write(r.rrdata)
				r.write(itemType{Type: nullAddressItem, Length: 0})
				r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + binary.Size(sr))})
				r.write(resp)
				r.write(sr)

			case ReadTag:
				p.debug("ReadTag")

				var (
					tagName  string
					tagCount uint16
				)

				if protd.PathSize > 0 && protdPath[0] == ansiExtended {
					tagName = string(protdPath[2 : protdPath[1]+2])
				}
				err = r.read(&tagCount)
				if err != nil {
					break loop
				}
				p.debug(tagName, tagCount)

				if rtData, tagType, ok := p.readTag(tagName, tagCount); ok {
					dataLen = uint16(binary.Size(resp)+binary.Size(tagType)) + typeLen(tagType)*tagCount
					addrLen = 0

					if cidok && r.connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(r.connID))
					}

					r.write(r.rrdata)
					if cidok && r.connID != 0 {
						r.write(itemType{Type: connAddressItem, Length: addrLen})
						r.write(r.connID)
						r.write(itemType{Type: connDataItem, Length: dataLen})
						r.write(protSeqCount)
					} else {
						r.write(itemType{Type: nullAddressItem, Length: addrLen})
						r.write(itemType{Type: unconnDataItem, Length: dataLen})
					}
					r.write(resp)
					r.write(tagType)
					r.write(rtData)

				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1
					addStatus := uint16(0)

					dataLen = uint16(binary.Size(resp) + binary.Size(addStatus))
					addrLen = 0

					if cidok && r.connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(r.connID))
					}

					r.write(r.rrdata)
					if cidok && r.connID != 0 {
						r.write(itemType{Type: connAddressItem, Length: addrLen})
						r.write(r.connID)
						r.write(itemType{Type: connDataItem, Length: dataLen})
						r.write(protSeqCount)
					} else {
						r.write(itemType{Type: nullAddressItem, Length: addrLen})
						r.write(itemType{Type: unconnDataItem, Length: dataLen})
					}
					r.write(resp)
					r.write(addStatus)
				}

			case WriteTag:
				p.debug("WriteTag")

				var (
					tagName  string
					tagType  uint16
					tagCount uint16
				)

				if protd.PathSize > 0 && protdPath[0] == ansiExtended {
					tagName = string(protdPath[2 : protdPath[1]+2])
				}
				err = r.read(&tagType)
				if err != nil {
					break loop
				}
				err = r.read(&tagCount)
				if err != nil {
					break loop
				}
				p.debug(tagName, tagType, tagCount)

				wrData := make([]uint8, typeLen(tagType)*tagCount)
				err = r.read(wrData)
				if err != nil {
					break loop
				}

				if p.saveTag(tagName, tagType, tagCount, wrData) {
					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if cidok && r.connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(r.connID))
					}

					r.write(r.rrdata)
					if cidok && r.connID != 0 {
						r.write(itemType{Type: connAddressItem, Length: addrLen})
						r.write(r.connID)
						r.write(itemType{Type: connDataItem, Length: dataLen})
						r.write(protSeqCount)
					} else {
						r.write(itemType{Type: nullAddressItem, Length: addrLen})
						r.write(itemType{Type: unconnDataItem, Length: dataLen})
					}
					r.write(resp)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1
					addStatus := uint16(0)

					dataLen = uint16(binary.Size(resp) + binary.Size(addStatus))
					addrLen = 0

					if cidok && r.connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(r.connID))
					}

					r.write(r.rrdata)
					if cidok && r.connID != 0 {
						r.write(itemType{Type: connAddressItem, Length: addrLen})
						r.write(r.connID)
						r.write(itemType{Type: connDataItem, Length: dataLen})
						r.write(protSeqCount)
					} else {
						r.write(itemType{Type: nullAddressItem, Length: addrLen})
						r.write(itemType{Type: unconnDataItem, Length: dataLen})
					}
					r.write(resp)
					r.write(addStatus)
				}

			case Reset:
				p.debug("Reset")

				if p.callback != nil {
					go p.callback(Reset, Success, nil)
				}

				r.write(r.rrdata)
				r.write(itemType{Type: nullAddressItem, Length: 0})
				r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				r.write(resp)

			default:
				p.debug("unknown service:", protd.Service)

				resp.Status = 0x08 // Service not supported

				r.write(r.rrdata)
				r.write(itemType{Type: nullAddressItem, Length: 0})
				r.write(itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				r.write(resp)
			}

		default:
			p.debug("unknown command:", r.encHead.Command)

			data := make([]uint8, r.encHead.Length)
			err = r.read(&data)
			if err != nil {
				break loop
			}
			r.encHead.Status = 0x01

			r.write(data)
		}

		err = conn.SetWriteDeadline(time.Now().Add(p.Timeout))
		if err != nil {
			fmt.Println(err)
			break loop
		}

		r.encHead.Length = uint16(r.writeBuf.Len())
		var buf bytes.Buffer

		err = binary.Write(&buf, binary.LittleEndian, r.encHead)
		if err != nil {
			fmt.Println(err)
			break loop
		}
		buf.Write(r.writeBuf.Bytes())

		_, err = conn.Write(buf.Bytes())
		if err != nil {
			fmt.Println(err)
			break loop
		}
	}
	err := conn.Close()
	if err != nil {
		fmt.Println(err)
	}
}
