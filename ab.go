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
	"time"
)

// PLC .
type PLC struct {
	callback  func(service int, statut int, tag *Tag)
	closeI    bool
	closeMut  sync.RWMutex
	closeWMut sync.Mutex
	closeWait *sync.Cond
	eds       map[string]map[string]string
	port      uint16
	symbols   *Class
	template  *Class
	tids      map[string]structData
	tidLast   int
	tMut      sync.RWMutex
	tags      map[string]*Tag

	Class       map[int]*Class
	DumpNetwork bool // enables dumping network packets
	Name        string
	Verbose     bool // enables debugging output
	Timeout     time.Duration
}

// Init initialize library. Must be called first.
func Init(eds string) (*PLC, error) {
	var p PLC
	p.Class = make(map[int]*Class)
	p.tags = make(map[string]*Tag)
	p.tids = make(map[string]structData)
	p.tidLast = 1
	p.Timeout = 60 * time.Second

	err := p.loadEDS(eds)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *PLC) debug(args ...interface{}) {
	if p.Verbose {
		fmt.Println(args...)
	}
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
	sock.Control = sockControl
	serv2, err := sock.Listen(context.Background(), "tcp", host)
	if err != nil {
		fmt.Println("plcconnector Serve: ", err)
		return err
	}
	p.port = getPort(host)
	serv := serv2.(*net.TCPListener)
	go p.serveUDP(host)
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
	file     map[int]*[3]uint8
	rrdata   sendData
	encHead  encapsulationHeader
	p        *PLC
	readBuf  *bufio.Reader
	writeBuf *bytes.Buffer
	wrCIPBuf *bytes.Buffer
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

func (r *req) writeCIP(data interface{}) {
	err := binary.Write(r.wrCIPBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func (r *req) reset() {
	r.readBuf.Reset(r.c)
	r.writeBuf.Reset()
	r.wrCIPBuf.Reset()
}

func (p *PLC) handleRequest(conn net.Conn) {
	r := req{}
	r.connID = uint32(0)
	r.c = conn
	r.file = make(map[int]*[3]uint8)
	r.p = p
	r.readBuf = bufio.NewReader(conn)
	r.writeBuf = new(bytes.Buffer)
	r.wrCIPBuf = new(bytes.Buffer)

loop:
	for {
		r.reset()

		p.closeMut.RLock()
		endP := p.closeI
		p.closeMut.RUnlock()
		if endP {
			break loop
		}

		timeout := time.Now().Add(p.Timeout)
		err := conn.SetReadDeadline(timeout)
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
		case ecNOP:
			if r.eipNOP() != nil {
				break loop
			}
			continue loop

		case ecRegisterSession:
			if r.eipRegisterSession() != nil {
				break loop
			}

		case ecUnRegisterSession:
			p.debug("UnregisterSession")
			break loop

		case ecListIdentity:
			if r.eipListIdentity() != nil {
				break loop
			}

		case ecListServices:
			if r.eipListServices() != nil {
				break loop
			}

		case ecListInterfaces:
			p.debug("ListInterfaces")
			r.write(uint16(0)) // ItemCount

		case ecSendRRData, ecSendUnitData:
			p.debug("SendRRData/SendUnitData")

			var (
				item         itemType
				protd        protocolData
				protSeqCount uint16
				resp         response
				dataLen      int
			)
			err = r.read(&r.rrdata)
			if err != nil {
				break loop
			}

			if r.rrdata.Timeout != 0 && r.encHead.Command == ecSendRRData {
				timeout = time.Now().Add(time.Duration(r.rrdata.Timeout) * time.Second)
				err = conn.SetReadDeadline(timeout)
				if err != nil {
					fmt.Println(err)
					break loop
				}
			}

			r.rrdata.Timeout = 0
			cidok := false
			mayCon := false
			itemserror := false

			if r.rrdata.ItemCount != 2 {
				p.debug("itemCount != 2")
				r.encHead.Status = eipIncorrectData
				break command
			}

			// address item
			err = r.read(&item)
			if err != nil {
				break loop
			}
			if item.Type == itConnAddress { // TODO itemdata to connID
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != itNullAddress {
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
			dataLen = int(item.Length)
			maxData := 65000
			if item.Type == itConnData {
				err = r.read(&protSeqCount)
				if err != nil {
					break loop
				}
				maxData = 472 // FIXME read from forward open
				dataLen -= 2
				cidok = true
			} else if item.Type != itUnconnData {
				p.debug("unkown data item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			if itemserror {
				r.encHead.Status = eipIncorrectData
				break command
			}

			// CIP
			err = r.read(&protd)
			if err != nil {
				break loop
			}

			resp.Service = protd.Service + 128
			resp.Status = Success

			ePath := make([]uint8, protd.PathSize*2)
			err = r.read(&ePath)
			if err != nil {
				break loop
			}
			dataLen -= 2 + len(ePath)

			class, instance, attr, path, err := r.parsePath(ePath)
			if p.Verbose {
				fmt.Printf("Class %X Instance %X Attr %X %v\n", class, instance, attr, path)
			}
			// if err != nil {
			// 	resp.Status = PathSegmentError
			// 	resp.AddStatusSize = 1

			// 	r.write(resp)
			// 	r.write(uint16(0))
			// 	break command // FIXME
			// }

			if class == ConnManager && protd.Service == UnconnectedSend {
				var usdata [4]uint8
				err = r.read(&usdata)
				if err != nil {
					break loop
				}
				err = r.read(&protd)
				if err != nil {
					break loop
				}

				resp.Service = protd.Service + 128

				ePath = make([]uint8, protd.PathSize*2)
				err = r.read(&ePath)
				if err != nil {
					break loop
				}
				dataLen -= 6 + len(ePath)

				class, instance, attr, path, err = r.parsePath(ePath)
				if p.Verbose {
					fmt.Printf("UNC SEND: Class %X Instance %X Attr %X %v\n", class, instance, attr, path)
				}
			}

			switch {
			case protd.Service == GetAttrAll:
				p.debug("GetAttributesAll")
				mayCon = true

				in := p.GetClassInstance(class, instance)
				if in != nil {
					r.write(resp)
					r.write(in.getAttrAll())
				} else {
					p.debug("path unknown", path)
					if class == FileClass {
						resp.Status = ObjectNotExist
					} else {
						resp.Status = PathUnknown
					}
					r.write(resp)
				}

			case protd.Service == GetAttrList:
				p.debug("GetAttributesList")
				mayCon = true
				var (
					count uint16
					buf   bytes.Buffer
					st    uint16
				)

				err = r.read(&count)
				if err != nil {
					break loop
				}
				attr := make([]uint16, count)
				err = r.read(&attr)
				if err != nil {
					break loop
				}

				in := p.GetClassInstance(class, instance)
				if in != nil {
					in.m.RLock()
					ln := len(in.attr)
					for _, i := range attr {
						bwrite(&buf, i)
						if int(i) < ln && in.attr[i] != nil {
							p.debug(in.attr[i].Name)
							st = Success
							bwrite(&buf, st)
							bwrite(&buf, in.attr[i].data)
						} else {
							resp.Status = AttrListError
							st = AttrNotSup
							bwrite(&buf, st)
						}
					}
					in.m.RUnlock()

					r.write(resp)
					r.write(count)
					r.write(buf.Bytes())
				} else {
					p.debug("path unknown", path)
					if class == FileClass {
						resp.Status = ObjectNotExist
					} else {
						resp.Status = PathUnknown
					}
					r.write(resp)
				}

			case protd.Service == GetInstAttrList:
				p.debug("GetInstanceAttributesList")
				mayCon = true
				var (
					count uint16
					buf   bytes.Buffer
				)

				err = r.read(&count)
				if err != nil {
					break loop
				}
				attr := make([]uint16, count)
				err = r.read(&attr)
				if err != nil {
					break loop
				}

				li, ins := p.GetClassInstancesList(class, instance)
				if li != nil {
					for a, x := range li {
						if buf.Len() >= maxData-20 {
							resp.Status = PartialTransfer
							break
						}
						bwrite(&buf, uint32(x))
						in := ins[a]
						in.m.RLock()
						ln := len(in.attr)
						for _, i := range attr {
							if int(i) < ln && in.attr[i] != nil {
								bwrite(&buf, in.attr[i].data)
							} else { // FIXME break
								resp.Status = AttrListError
							}
						}
						in.m.RUnlock()
					}

					r.write(resp)
					r.write(buf.Bytes())
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case protd.Service == GetAttr:
				p.debug("GetAttributesSingle")
				mayCon = true

				var (
					aok bool
					at  *Tag
				)
				in := p.GetClassInstance(class, instance)
				if in != nil {
					in.m.RLock()
					if attr < len(in.attr) {
						at = in.attr[attr]
						if at != nil {
							aok = true
						}
					}
					in.m.RUnlock()
				}
				resp.Service = protd.Service + 128

				if in != nil && aok {
					p.debug(at.Name)
					r.write(resp)
					r.write(at.data)
				} else {
					p.debug("path unknown", path)
					if class == FileClass {
						resp.Status = ObjectNotExist
					} else {
						resp.Status = PathUnknown
					}
					r.write(resp)
				}

			case class == FileClass && protd.Service == InititateUpload:
				p.debug("InititateUpload")
				mayCon = true
				var maxSize uint8

				err = r.read(&maxSize)
				if err != nil {
					break loop
				}

				in := p.GetClassInstance(class, instance)
				if in != nil {
					var sr initUploadResponse
					sr.FileSize = uint32(len(in.data))
					sr.TransferSize = maxSize
					r.file[instance] = &[3]uint8{maxSize, 0, 0} // TransferSize, TransferNumber, TransferNumber rollover
					r.write(resp)
					r.write(sr)
				} else {
					p.debug("path unknown", path)
					resp.Status = PathUnknown
					r.write(resp)
				}

			case class == FileClass && protd.Service == UploadTransfer:
				p.debug("UploadTransfer")
				mayCon = true
				var transferNo uint8

				err = r.read(&transferNo)
				if err != nil {
					break loop
				}

				in := p.GetClassInstance(class, instance)
				f, fok := r.file[instance]
				if in != nil && fok {
					if transferNo == f[1] || transferNo == f[1]+1 || (transferNo == 0 && f[1] == 255) {
						if transferNo == 0 && f[1] == 255 { // rollover
							p.debug("rollover")
							f[2]++ // FIXME retry!
						}

						var sr uploadTransferResponse
						addcksum := false
						dtlen := len(in.data)
						pos := (int(f[2]) + 1) * int(transferNo) * int(f[0])
						posto := pos + int(f[0])
						if posto > dtlen {
							posto = dtlen
						}
						dt := in.data[pos:posto]
						sr.TransferNumber = transferNo
						if transferNo == 0 && dtlen <= int(f[0]) {
							sr.TranferPacketType = tptFirstLast
							addcksum = true
						} else if transferNo == 0 && f[2] == 0 {
							sr.TranferPacketType = tptFirst
						} else if pos+int(f[0]) >= dtlen {
							sr.TranferPacketType = tptLast
							addcksum = true
						} else {
							sr.TranferPacketType = tptMiddle
						}
						f[1] = transferNo

						ln := uint16(binary.Size(resp) + binary.Size(sr) + len(dt))
						if addcksum {
							ln += uint16(binary.Size(in.getAttrData(7)))
						}

						p.debug(pos, ":", posto)

						r.write(resp)
						r.write(sr)
						r.write(dt)
						if addcksum {
							r.write(in.getAttrData(7))
						}
					} else {
						p.debug("transfer number error", transferNo)

						resp.Status = InvalidPar
						resp.AddStatusSize = 1

						r.write(resp)
						r.write(uint16(0))
					}
				} else {
					p.debug("path unknown", path)

					resp.Status = PathUnknown
					r.write(resp)
				}

			case class == ConnManager && protd.Service == ForwardOpen:
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

				r.write(resp)
				r.write(sr)

			case class == ConnManager && protd.Service == ForwardClose:
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

				r.write(resp)
				r.write(sr)

			case class == TemplateClass && protd.Service == ReadTemplate: // TODO Status 0x06
				p.debug("ReadTemplate")
				mayCon = true

				var rd readTemplateResponse

				err = r.read(&rd)
				if err != nil {
					break loop
				}
				p.debug(rd.Offset, rd.Number)

				if in := p.GetClassInstance(class, instance); in != nil {
					r.write(resp)
					r.write(in.data)
				} else {
					p.debug("path unknown", path)

					resp.Status = PathUnknown
					r.write(resp)
				}

			case protd.Service == ReadTag:
				p.debug("ReadTag")
				mayCon = true

				var tagCount uint16

				err = r.read(&tagCount)
				if err != nil {
					break loop
				}

				if rtData, tagType, elLen, ok := p.readTag(path, tagCount); ok {
					if tagType >= TypeStructHead {
						r.write(uint16(tagType >> 16))
					}
					if len(rtData) > maxData {
						resp.Status = PartialTransfer
						rtData = rtData[:(maxData/elLen)*elLen]
					}
					r.write(resp)
					r.write(uint16(tagType))
					r.write(rtData)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case protd.Service == ReadTagFrag:
				p.debug("ReadTagFragmented")
				mayCon = true

				var (
					tagCount  uint16
					tagOffset uint32
				)

				err = r.read(&tagCount)
				if err != nil {
					break loop
				}
				err = r.read(&tagOffset)
				if err != nil {
					break loop
				}

				if rtData, tagType, elLen, ok := p.readTag(path, tagCount); ok && tagOffset < uint32(len(rtData)) {
					if tagType >= TypeStructHead {
						r.write(uint16(tagType >> 16))
					}
					rtData = rtData[tagOffset:]
					if len(rtData) > maxData {
						resp.Status = PartialTransfer
						rtData = rtData[:(maxData/elLen)*elLen]
					}
					r.write(resp)
					r.write(uint16(tagType))
					r.write(rtData)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case protd.Service == ReadModifyWrite:
				p.debug("ReadModifyWrite")
				mayCon = true

				var maskSize uint16

				err = r.read(&maskSize)
				if err != nil {
					break loop
				}
				orMask := make([]uint8, maskSize)
				err = r.read(&orMask)
				if err != nil {
					break loop
				}
				andMask := make([]uint8, maskSize)
				err = r.read(&andMask)
				if err != nil {
					break loop
				}
				if p.readModWriteTag(path, orMask, andMask) {
					r.write(resp)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case protd.Service == WriteTag:
				p.debug("WriteTag")
				mayCon = true

				var (
					tagType  uint16
					tagCount uint16
				)

				err = r.read(&tagType)
				if err != nil {
					break loop
				}
				err = r.read(&tagCount)
				if err != nil {
					break loop
				}

				wrData := make([]uint8, typeLen(tagType)*tagCount)
				err = r.read(wrData)
				if err != nil {
					break loop
				}

				if p.saveTag(path, tagType, int(tagCount), wrData, 0) {
					r.write(resp)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case protd.Service == WriteTagFrag:
				p.debug("WriteTagFragmented")
				mayCon = true

				var (
					tagType   uint16
					tagCount  uint16
					tagOffset uint32
				)

				err = r.read(&tagType)
				if err != nil {
					break loop
				}
				err = r.read(&tagCount)
				if err != nil {
					break loop
				}
				err = r.read(&tagOffset)
				if err != nil {
					break loop
				}

				wrData := make([]uint8, dataLen-8)
				err = r.read(wrData)
				if err != nil {
					break loop
				}

				if p.saveTag(path, tagType, (dataLen-8)/int(typeLen(tagType)), wrData, int(tagOffset)) {
					r.write(resp)
				} else {
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1

					r.write(resp)
					r.write(uint16(0))
				}

			case protd.Service == Reset:
				p.debug("Reset")

				if p.callback != nil {
					go p.callback(Reset, Success, nil)
				}
				r.write(resp)

			default:
				fmt.Println("unknown service:", protd.Service)

				resp.Status = ServNotSup
				r.write(resp)
			}

			r.writeCIP(r.rrdata)
			if mayCon && cidok && r.connID != 0 {
				r.writeCIP(itemType{Type: itConnAddress, Length: uint16(binary.Size(r.connID))})
				r.writeCIP(r.connID)
				r.writeCIP(itemType{Type: itConnData, Length: uint16(binary.Size(protSeqCount) + r.writeBuf.Len())})
				r.writeCIP(protSeqCount)
			} else {
				r.writeCIP(itemType{Type: itNullAddress, Length: 0})
				r.writeCIP(itemType{Type: itUnconnData, Length: uint16(r.writeBuf.Len())})
			}

		default:
			fmt.Println("unknown command:", r.encHead.Command)

			data := make([]uint8, r.encHead.Length)
			err = r.read(&data)
			if err != nil {
				break loop
			}
			r.encHead.Status = eipInvalid

			r.write(data)
		}

		unread := r.readBuf.Buffered()
		if unread > 0 {
			discard := make([]byte, unread)
			r.read(&discard)
			p.debug("DISCARDED:", discard)
		}

		err = conn.SetWriteDeadline(timeout)
		if err != nil {
			fmt.Println(err)
			break loop
		}

		r.encHead.Length = uint16(r.wrCIPBuf.Len() + r.writeBuf.Len())
		var buf bytes.Buffer

		err = binary.Write(&buf, binary.LittleEndian, r.encHead)
		if err != nil {
			fmt.Println(err)
			break loop
		}
		buf.Write(r.wrCIPBuf.Bytes())
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
