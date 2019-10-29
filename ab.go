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
	"io"
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
func Init(testTags bool) *PLC {
	var p PLC
	p.Class = make(map[int]Class)
	p.tags = make(map[string]*Tag)
	p.Timeout = 60 * time.Second

	p.Class[1] = defaultIdentityClass()

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

	return &p
}

func (p *PLC) debug(args ...interface{}) {
	if p.Verbose {
		fmt.Println(args...)
	}
}

func (p *PLC) readData(r io.Reader, data interface{}) error {
	err := binary.Read(r, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	if p.DumpNetwork {
		fmt.Printf("%#v\n", data)
	}
	return err
}

func (p *PLC) writeData(w io.Writer, data interface{}) {
	err := binary.Write(w, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
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

func (p *PLC) handleRequest(conn net.Conn) {
	connID := uint32(0)

	readBuf := bufio.NewReader(conn)
	writeBuf := new(bytes.Buffer)

loop:
	for {
		readBuf.Reset(conn)
		writeBuf.Reset()

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
		var encHead encapsulationHeader
		err = p.readData(readBuf, &encHead)
		if err != nil {
			break loop
		}

	command:
		switch encHead.Command {
		case nop:
			p.debug("NOP")

			data := make([]byte, encHead.Length)
			err = p.readData(readBuf, &data)
			if err != nil {
				break loop
			}
			continue loop

		case registerSession:
			p.debug("RegisterSession")

			var data registerSessionData
			err = p.readData(readBuf, &data)
			if err != nil {
				break loop
			}

			encHead.SessionHandle = rand.Uint32()

			p.writeData(writeBuf, encHead)
			p.writeData(writeBuf, data)

		case unregisterSession:
			p.debug("UnregisterSession")
			break loop

		case listIdentity: // UDP!
			p.debug("ListIdentity")

			itemCount := uint16(1)
			state := uint8(0)
			productName := []byte{77, 111, 110, 103, 111, 108, 80, 76, 67}
			var (
				data listIdentityData
				typ  itemType
			)

			data.ProtocolVersion = 1
			data.SocketFamily = htons(2)
			data.SocketPort = htons(p.port)
			data.SocketAddr = getIP4()
			data.VendorID = 1
			data.DeviceType = 0x0C // communications adapter
			data.ProductCode = 65001
			data.Revision[0] = 1
			data.Revision[1] = 0
			data.Status = 0 // Owned
			data.SerialNumber = 1
			data.ProductNameLength = uint8(len(productName))

			typ.Type = 0x0C
			typ.Length = uint16(binary.Size(data) + len(productName) + binary.Size(state))

			encHead.Length = uint16(binary.Size(itemCount) + binary.Size(typ) + int(typ.Length))
			p.writeData(writeBuf, encHead)
			p.writeData(writeBuf, itemCount)
			p.writeData(writeBuf, typ)
			p.writeData(writeBuf, data)
			p.writeData(writeBuf, productName)
			p.writeData(writeBuf, state)

		case listServices:
			p.debug("ListServices")

			itemCount := uint16(1)
			var (
				data listServicesData
				typ  itemType
			)

			typ.Type = cipItemIDListServiceResponse
			typ.Length = uint16(binary.Size(data))

			data.ProtocolVersion = 1
			data.CapabilityFlags = capabilityFlagsCipTCP
			data.NameOfService = [16]int8{67, 111, 109, 109, 117, 110, 105, 99, 97, 116, 105, 111, 110, 115, 0, 0} // Communications

			encHead.Length = uint16(binary.Size(itemCount) + binary.Size(typ) + binary.Size(data))
			p.writeData(writeBuf, encHead)
			p.writeData(writeBuf, itemCount)
			p.writeData(writeBuf, typ)
			p.writeData(writeBuf, data)

		case listInterfaces:
			p.debug("ListInterfaces")

			itemCount := uint16(0)

			encHead.Length = uint16(binary.Size(itemCount))
			p.writeData(writeBuf, encHead)
			p.writeData(writeBuf, itemCount)

		case sendRRData, sendUnitData:
			p.debug("SendRRData/SendUnitData")

			var (
				data         sendData
				item         itemType
				dataLen      uint16
				addrLen      uint16
				protd        protocolData
				protSeqCount uint16
			)
			err = p.readData(readBuf, &data)
			if err != nil {
				break loop
			}

			data.Timeout = 0
			cidok := false
			itemserror := false

			if data.ItemCount != 2 {
				p.debug("itemCount != 2")
				encHead.Length = 0
				encHead.Status = 0x03 // Incorrect data
				p.writeData(writeBuf, encHead)
				break command
			}

			// address item
			err = p.readData(readBuf, &item)
			if err != nil {
				break loop
			}
			if item.Type == connAddressItem { // TODO itemdata to connID
				itemdata := make([]uint8, item.Length)
				err = p.readData(readBuf, &itemdata)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != nullAddressItem {
				p.debug("unkown address item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = p.readData(readBuf, &itemdata)
				if err != nil {
					break loop
				}
			}

			// data item
			err = p.readData(readBuf, &item)
			if err != nil {
				break loop
			}
			if item.Type == connDataItem {
				err = p.readData(readBuf, &protSeqCount)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != unconnDataItem {
				p.debug("unkown data item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = p.readData(readBuf, &itemdata)
				if err != nil {
					break loop
				}
			}

			if itemserror {
				encHead.Length = 0
				encHead.Status = 0x03 // Incorrect data
				p.writeData(writeBuf, encHead)
				break command
			}

			// CIP
			err = p.readData(readBuf, &protd)
			if err != nil {
				break loop
			}

			protdPath := make([]uint8, protd.PathSize*2)
			err = p.readData(readBuf, &protdPath)
			if err != nil {
				break loop
			}

			switch protd.Service {
			case GetAttrAll:
				p.debug("GetAttributesAll")
				var (
					resp response
					iok  bool
					in   Instance
				)
				c, cok := p.Class[int(protdPath[1])]
				if cok {
					in, iok = c.Inst[int(protdPath[3])]
				}
				resp.Service = protd.Service + 128

				if cok && iok {
					p.debug(c.Name)

					attrdata, attrlen := in.getAttrAll()

					encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp) + attrlen)
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
					p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + attrlen)})
					p.writeData(writeBuf, resp)
					p.writeData(writeBuf, attrdata)
				} else {
					p.debug("path unknown", protdPath)

					resp.Status = 0x05 // Path destination unknown

					encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
					p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
					p.writeData(writeBuf, resp)
				}

			case GetAttr:
				p.debug("GetAttributesSingle")

				var (
					resp response
					iok  bool
					in   Instance
					aok  bool
					at   Attribute
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

					encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp) + len(at.data))
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
					p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp) + len(at.data))})
					p.writeData(writeBuf, resp)
					p.writeData(writeBuf, at.data)
				} else {
					p.debug("path unknown", protdPath)

					resp.Status = 0x05 // Path destination unknown

					encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
					p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
					p.writeData(writeBuf, resp)
				}

			case ForwardOpen:
				p.debug("ForwardOpen")

				var (
					fodata forwardOpenData
					resp   forwardOpenResponse
				)
				err = p.readData(readBuf, &fodata)
				if err != nil {
					break loop
				}
				connPath := make([]uint8, fodata.ConnPathSize*2)
				err = p.readData(readBuf, &connPath)
				if err != nil {
					break loop
				}

				resp.Service = ForwardOpen | 128
				resp.Status = 0
				resp.OTConnectionID = rand.Uint32()
				resp.TOConnectionID = fodata.TOConnectionID
				resp.ConnSerialNumber = fodata.ConnSerialNumber
				resp.VendorID = fodata.VendorID
				resp.OriginatorSerialNumber = fodata.OriginatorSerialNumber
				resp.OTAPI = fodata.OTRPI
				resp.TOAPI = fodata.TORPI
				resp.AppReplySize = 0

				connID = fodata.TOConnectionID

				encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
				p.writeData(writeBuf, encHead)
				p.writeData(writeBuf, data)
				p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				p.writeData(writeBuf, resp)

			case ForwardClose:
				p.debug("ForwardClose")

				var (
					fcdata forwardCloseData
					resp   forwardCloseResponse
				)
				err = p.readData(readBuf, &fcdata)
				if err != nil {
					break loop
				}
				connPath := make([]uint8, fcdata.ConnPathSize*2)
				err = p.readData(readBuf, &connPath)
				if err != nil {
					break loop
				}

				resp.Service = ForwardClose | 128
				resp.Status = 0
				resp.ConnSerialNumber = fcdata.ConnSerialNumber
				resp.VendorID = fcdata.VendorID
				resp.OriginatorSerialNumber = fcdata.OriginatorSerialNumber
				resp.AppReplySize = 0

				connID = 0

				encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
				p.writeData(writeBuf, encHead)
				p.writeData(writeBuf, data)
				p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				p.writeData(writeBuf, resp)

			case ReadTag:
				p.debug("ReadTag")

				var (
					tagName  string
					tagCount uint16
				)

				if protd.PathSize > 0 && protdPath[0] == ansiExtended {
					tagName = string(protdPath[2 : protdPath[1]+2])
				}
				err = p.readData(readBuf, &tagCount)
				if err != nil {
					break loop
				}
				p.debug(tagName, tagCount)

				if rtData, rtType, ok := p.readTag(tagName, tagCount); ok {
					var resp readTagResponse

					resp.Service = ReadTag | 128
					resp.Status = Success
					resp.TagType = rtType

					dataLen = uint16(binary.Size(resp)) + typeLen(resp.TagType)*tagCount
					addrLen = 0

					if cidok && connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					if cidok && connID != 0 {
						p.writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						p.writeData(writeBuf, connID)
						p.writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						p.writeData(writeBuf, protSeqCount)
					} else {
						p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					p.writeData(writeBuf, resp)
					p.writeData(writeBuf, rtData)

				} else {
					var resp errorResponse

					resp.Service = ReadTag | 128
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1
					resp.AddStatus = 0

					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if cidok && connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					if cidok && connID != 0 {
						p.writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						p.writeData(writeBuf, connID)
						p.writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						p.writeData(writeBuf, protSeqCount)
					} else {
						p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					p.writeData(writeBuf, resp)
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
				err = p.readData(readBuf, &tagType)
				if err != nil {
					break loop
				}
				err = p.readData(readBuf, &tagCount)
				if err != nil {
					break loop
				}
				p.debug(tagName, tagType, tagCount)

				wrData := make([]uint8, typeLen(tagType)*tagCount)
				err = p.readData(readBuf, wrData)
				if err != nil {
					break loop
				}

				if p.saveTag(tagName, tagType, tagCount, wrData) {
					var resp response

					resp.Service = WriteTag | 128
					resp.Status = Success

					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if cidok && connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					if cidok && connID != 0 {
						p.writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						p.writeData(writeBuf, connID)
						p.writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						p.writeData(writeBuf, protSeqCount)
					} else {
						p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					p.writeData(writeBuf, resp)
				} else {
					var resp errorResponse

					resp.Service = WriteTag | 128
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1
					resp.AddStatus = 0

					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if cidok && connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					p.writeData(writeBuf, encHead)
					p.writeData(writeBuf, data)
					if cidok && connID != 0 {
						p.writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						p.writeData(writeBuf, connID)
						p.writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						p.writeData(writeBuf, protSeqCount)
					} else {
						p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					p.writeData(writeBuf, resp)
				}

			case Reset:
				p.debug("Reset")

				var resp response

				resp.Service = Reset + 128

				if p.callback != nil {
					go p.callback(Reset, Success, nil)
				}

				encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
				p.writeData(writeBuf, encHead)
				p.writeData(writeBuf, data)
				p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				p.writeData(writeBuf, resp)

			default:
				p.debug("unknown service:", protd.Service)
				var resp response

				resp.Service = protd.Service + 128
				resp.Status = 0x08 // Service not supported

				encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
				p.writeData(writeBuf, encHead)
				p.writeData(writeBuf, data)
				p.writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				p.writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				p.writeData(writeBuf, resp)
			}

		default:
			p.debug("unknown command:", encHead.Command)

			data := make([]uint8, encHead.Length)
			err = p.readData(readBuf, &data)
			if err != nil {
				break loop
			}
			encHead.Status = 0x01

			p.writeData(writeBuf, encHead)
			p.writeData(writeBuf, data)
		}

		err = conn.SetWriteDeadline(time.Now().Add(p.Timeout))
		if err != nil {
			fmt.Println(err)
			break loop
		}

		_, err = conn.Write(writeBuf.Bytes())
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
