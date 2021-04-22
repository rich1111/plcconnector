// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Copyright 2020 github.com/podeszfa All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package plcconnector implements communication with PLC.
package plcconnector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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
	timOff    time.Duration

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
		return err
	}
	p.port = getPort(host)
	serv := serv2.(*net.TCPListener)
	go p.serveUDP(host)
	for {
		err = serv.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			return err
		}
		conn, err := serv.AcceptTCP()
		if e, ok := err.(net.Error); ok && e.Timeout() {
			p.closeMut.RLock()
			endP := p.closeI
			p.closeMut.RUnlock()
			if endP {
				break
			}
		} else if err != nil {
			return err
		} else {
			go p.handleRequest(conn)
		}
	}
	err = serv.Close()
	if err != nil {
		return err
	}
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
	class    int
	instance int
	attr     int
	member   int
	path     []pathEl

	c        net.Conn
	connID   uint32
	dataLen  int
	lenRem   int
	uDataLen int
	encHead  encapsulationHeader
	file     map[int]*[3]uint8
	maxData  int
	maxFO    int
	p        *PLC
	protd    protocolData
	readBuf  *bufio.Reader
	resp     response
	rrdata   sendData
	wrCIPBuf *bytes.Buffer
	writeBuf *bytes.Buffer
}

func (r *req) read(data interface{}) (bool, error) {
	toRead := binary.Size(data)
	if r.lenRem != -1 && r.lenRem < toRead {
		r.err(NotEnoughData)
		return true, errors.New("not enough data")
	}
	err := binary.Read(r.readBuf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	r.lenRem -= toRead
	if r.p.DumpNetwork {
		fmt.Printf("%#v\n", data)
	}
	return false, err
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
	r.lenRem = -1
	r.writeBuf.Reset()
	r.wrCIPBuf.Reset()
}

func (r *req) err(status int) bool {
	r.resp.Status = uint8(status)
	r.write(r.resp)
	return true
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
	r.maxFO = 472

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
		_, err = r.read(&r.encHead)
		if err != nil {
			break loop
		}
		r.lenRem = int(r.encHead.Length)

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
				protSeqCount uint16
			)
			_, err = r.read(&r.rrdata)
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
			itemserror := false

			if r.rrdata.ItemCount != 2 {
				p.debug("itemCount != 2")
				r.encHead.Status = eipIncorrectData
				break
			}

			// address item
			_, err = r.read(&item)
			if err != nil {
				break loop
			}
			if item.Type == itConnAddress { // TODO itemdata to connID
				itemdata := make([]uint8, item.Length)
				_, err = r.read(&itemdata)
				if err != nil {
					break loop
				}
				cidok = true
			} else if item.Type != itNullAddress {
				p.debug("unkown address item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				_, err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			// data item
			_, err = r.read(&item)
			if err != nil {
				break loop
			}
			r.dataLen = int(item.Length)
			r.maxData = 472
			if item.Type == itConnData {
				_, err = r.read(&protSeqCount)
				if err != nil {
					break loop
				}
				r.maxData = r.maxFO
				r.dataLen -= 2
				cidok = true
			} else if item.Type != itUnconnData {
				p.debug("unkown data item:", item.Type)
				itemserror = true
				itemdata := make([]uint8, item.Length)
				_, err = r.read(&itemdata)
				if err != nil {
					break loop
				}
			}

			if itemserror {
				r.encHead.Status = eipIncorrectData
				break
			}

			// CIP
			_, err = r.read(&r.protd)
			if err != nil {
				break loop
			}

			r.resp.Service = r.protd.Service + 128
			r.resp.Status = Success
			r.resp.AddStatusSize = 0

			ePath := make([]uint8, r.protd.PathSize*2)
			_, err = r.read(&ePath)
			if err != nil {
				break loop
			}
			r.dataLen -= 2 + len(ePath)
			r.uDataLen = r.dataLen

			unc := false

			r.class, r.instance, r.attr, r.member, r.path, err = r.parsePath(ePath)
			if err != nil {
				r.resp.Status = PathSegmentError
				r.resp.AddStatusSize = 1
				r.write(r.resp)
				r.write(uint16(0))
				r.readBuf.Reset(r.c)
				goto errl
			}
			if p.Verbose {
				fmt.Printf("Class %X Instance %X Attr %X %v\n", r.class, r.instance, r.attr, r.path)
			}

			if r.class == ConnManager && r.protd.Service == UnconnectedSend {
				unc = true
				var usdata itemType
				_, err = r.read(&usdata)
				if err != nil {
					break loop
				}
				_, err = r.read(&r.protd)
				if err != nil {
					break loop
				}

				r.resp.Service = r.protd.Service + 128

				ePath = make([]uint8, r.protd.PathSize*2)
				_, err = r.read(&ePath)
				if err != nil {
					break loop
				}
				r.uDataLen -= binary.Size(usdata) + int(usdata.Length)
				r.dataLen -= 6 + len(ePath) + r.uDataLen

				r.class, r.instance, r.attr, r.member, r.path, err = r.parsePath(ePath)
				if err != nil {
					r.resp.Status = PathSegmentError
					r.resp.AddStatusSize = 1
					r.write(r.resp)
					r.write(uint16(0))
					r.readBuf.Reset(r.c)
					goto errl
				}
				if p.Verbose {
					fmt.Printf("UNC SEND: Class %X Instance %X Attr %X %v\n", r.class, r.instance, r.attr, r.path)
				}
			}

			if !r.serviceHandle() {
				r.readBuf.Reset(r.c)
				break loop
			}

			if unc { // path
				// fmt.Println(">>>", r.uDataLen)
				data := make([]uint8, r.uDataLen)
				_, err = r.read(&data)
				if err != nil {
					break loop
				}
			}

		errl:
			r.writeCIP(r.rrdata)
			if cidok && r.connID != 0 {
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
			_, err = r.read(&data)
			if err != nil {
				break loop
			}
			r.encHead.Status = eipInvalid

			r.write(data)
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

func (r *req) serviceHandle() bool {
	switch {
	case r.class == MessageRouter && r.protd.Service == MultiServ: // TODO errors, status 6
		r.p.debug("MultipleServicePacket")

		var (
			count  uint16
			offset uint16
		)

		rb, err := r.read(&count)
		if err != nil {
			return rb
		}
		offset = 2 + 2*count

		svs := make([]uint16, count)
		rb, err = r.read(&svs)
		if err != nil {
			return rb
		}

		r.write(r.resp)
		r.write(count)

		oldBuf := r.writeBuf
		newBuf := new(bytes.Buffer)
		r.writeBuf = newBuf

		olddl := r.dataLen
		for i := range svs {
			rb, err = r.read(&r.protd)
			if err != nil {
				return rb
			}

			r.resp.Service = r.protd.Service + 128
			r.resp.Status = Success

			ePath := make([]uint8, r.protd.PathSize*2)
			rb, err = r.read(&ePath)
			if err != nil {
				return rb
			}
			if i+1 < len(svs) {
				r.dataLen = int(svs[i+1] - svs[i])
			} else {
				r.dataLen = olddl - int(svs[i])
			}
			r.dataLen -= 2 + len(ePath)

			r.class, r.instance, r.attr, r.member, r.path, err = r.parsePath(ePath)
			if r.p.Verbose {
				fmt.Printf("Class %X Instance %X Attr %X %v\n", r.class, r.instance, r.attr, r.path)
			}

			svs[i] = offset + uint16(r.writeBuf.Len())
			if !r.serviceHandle() {
				return false
			}
		}
		r.writeBuf = oldBuf
		r.write(svs)
		r.write(newBuf.Bytes())

	case r.protd.Service == GetAttrAll:
		r.p.debug("GetAttributesAll")

		in := r.p.GetClassInstance(r.class, r.instance)
		if in != nil {
			r.write(r.resp)
			r.write(in.getAttrAll())
		} else {
			r.p.debug("path unknown", r.path)
			if r.class == FileClass {
				r.resp.Status = ObjectNotExist
			} else {
				r.resp.Status = PathUnknown
			}
			r.write(r.resp)
		}

	case r.protd.Service == GetAttrList:
		r.p.debug("GetAttributesList")
		var (
			count uint16
			buf   bytes.Buffer
			st    uint16
		)

		rb, err := r.read(&count)
		if err != nil {
			return rb
		}
		attr := make([]uint16, count)
		rb, err = r.read(&attr)
		if err != nil {
			return rb
		}

		in := r.p.GetClassInstance(r.class, r.instance)
		if in != nil {
			in.m.RLock()
			ln := len(in.attr)
			for _, i := range attr {
				bwrite(&buf, i)
				if int(i) < ln && in.attr[i] != nil {
					r.p.debug(in.attr[i].Name)
					st = Success
					bwrite(&buf, st)
					bwrite(&buf, in.attr[i].DataBytes())
				} else {
					r.resp.Status = AttrListError
					st = AttrNotSup
					bwrite(&buf, st)
				}
			}
			in.m.RUnlock()

			r.write(r.resp)
			r.write(count)
			r.write(buf.Bytes())
		} else {
			r.p.debug("path unknown", r.path)
			if r.class == FileClass {
				r.resp.Status = ObjectNotExist
			} else {
				r.resp.Status = PathUnknown
			}
			r.write(r.resp)
		}

	case r.protd.Service == SetAttrList:
		r.p.debug("SetAttributesList")
		var (
			attr  uint16
			count uint16
			buf   bytes.Buffer
			st    uint16
		)

		rb, err := r.read(&count)
		if err != nil {
			return rb
		}

		in := r.p.GetClassInstance(r.class, r.instance)
		if in != nil {
			in.m.RLock()
			ln := len(in.attr)
			for i := uint16(0); i < count; i++ {
				rb, err := r.read(&attr)
				if err != nil {
					return rb
				}
				bwrite(&buf, attr)
				if int(attr) < ln && in.attr[attr] != nil {
					r.p.debug(in.attr[attr].Name)
					wrData := make([]uint8, len(in.attr[attr].data))
					rb, err := r.read(wrData)
					if err != nil {
						return rb
					}
					st = Success
					sdb := in.attr[attr].SetDataBytes(wrData)
					if sdb != Success {
						r.resp.Status = AttrListError
						st = uint16(sdb)
					}
				} else {
					r.resp.Status = AttrListError
					st = AttrNotSup
				}
				bwrite(&buf, st)
			}
			in.m.RUnlock()

			r.write(r.resp)
			r.write(count)
			r.write(buf.Bytes())
		} else {
			r.p.debug("path unknown", r.path)
			if r.class == FileClass {
				r.resp.Status = ObjectNotExist
			} else {
				r.resp.Status = PathUnknown
			}
			r.write(r.resp)
		}

	case r.class == SymbolClass && r.protd.Service == GetInstAttrList:
		r.p.debug("GetInstanceAttributesList")
		var (
			count uint16
			buf   bytes.Buffer
		)

		rb, err := r.read(&count)
		if err != nil {
			return rb
		}
		attr := make([]uint16, count)
		rb, err = r.read(&attr)
		if err != nil {
			return rb
		}

		li, ins := r.p.GetClassInstancesList(r.class, r.instance, 0)
		if li != nil {
			for a, x := range li {
				if buf.Len() >= r.maxData-20 {
					r.resp.Status = PartialTransfer
					break
				}
				bwrite(&buf, uint32(x))
				in := ins[a]
				in.m.RLock()
				ln := len(in.attr)
				for _, i := range attr {
					if int(i) < ln && in.attr[i] != nil {
						bwrite(&buf, in.attr[i].DataBytes())
					} else { // FIXME break
						r.resp.Status = AttrListError
					}
				}
				in.m.RUnlock()
			}

			r.write(r.resp)
			r.write(buf.Bytes())
		} else {
			r.err(PathUnknown)
		}

	case r.protd.Service == GetAttr:
		r.p.debug("GetAttributesSingle")

		at, aok, in := r.p.GetClassInstanceAttr(r.class, r.instance, r.attr)

		r.resp.Service = r.protd.Service + 128

		if in && aok {
			r.p.debug(at.Name)
			r.write(r.resp)
			r.write(at.DataBytes())
		} else {
			r.p.debug("path unknown", r.path)
			if in {
				r.resp.Status = AttrNotSup
			} else if r.class == FileClass {
				r.resp.Status = ObjectNotExist
			} else {
				r.resp.Status = PathUnknown
			}
			r.write(r.resp)
		}

	case r.protd.Service == SetAttr:
		r.p.debug("SetAttributesSingle")

		var (
			aok bool
			at  *Tag
		)

		wrData := make([]uint8, r.dataLen)
		rb, err := r.read(wrData)
		if err != nil {
			return rb
		}

		at, aok, in := r.p.GetClassInstanceAttr(r.class, r.instance, r.attr)

		r.resp.Service = r.protd.Service + 128

		if in && aok {
			r.p.debug(at.Name)
			if r.instance == 0 {
				r.resp.Status = ServNotSup
			} else {
				r.resp.Status = at.SetDataBytes(wrData)
			}
		} else {
			r.p.debug("path unknown", r.path)
			if in {
				if r.instance == 0 {
					r.resp.Status = ServNotSup
				} else {
					r.resp.Status = AttrNotSup
				}
			} else if r.class == FileClass {
				r.resp.Status = ObjectNotExist
			} else {
				r.resp.Status = PathUnknown
			}
		}
		r.write(r.resp)

	case r.class == FileClass && r.protd.Service == InititateUpload:
		r.p.debug("InititateUpload")
		var maxSize uint8

		rb, err := r.read(&maxSize)
		if err != nil {
			return rb
		}

		in := r.p.GetClassInstance(r.class, r.instance)
		if in != nil {
			var sr initUploadResponse
			sr.FileSize = uint32(len(in.data))
			sr.TransferSize = maxSize
			r.file[r.instance] = &[3]uint8{maxSize, 0, 0} // TransferSize, TransferNumber, TransferNumber rollover
			r.write(r.resp)
			r.write(sr)
		} else {
			r.err(PathUnknown)
		}

	case r.class == FileClass && r.protd.Service == UploadTransfer:
		r.p.debug("UploadTransfer")
		var transferNo uint8

		rb, err := r.read(&transferNo)
		if err != nil {
			return rb
		}

		in := r.p.GetClassInstance(r.class, r.instance)
		f, fok := r.file[r.instance]
		if in != nil && fok {
			if transferNo == f[1] || transferNo == f[1]+1 || (transferNo == 0 && f[1] == 255) {
				if transferNo == 0 && f[1] == 255 { // rollover
					r.p.debug("rollover")
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

				r.p.debug(pos, ":", posto)

				r.write(r.resp)
				r.write(sr)
				r.write(dt)
				if addcksum {
					r.write(in.getAttrData(7))
				}
			} else {
				r.p.debug("transfer number error", transferNo)

				r.resp.Status = InvalidPar
				r.resp.AddStatusSize = 1

				r.write(r.resp)
				r.write(uint16(0))
			}
		} else {
			r.err(PathUnknown)
		}

	case r.class == ConnManager && r.protd.Service == ForwardOpen:
		r.p.debug("ForwardOpen")

		var (
			fodata forwardOpenData
			sr     forwardOpenResponse
		)

		rb, err := r.read(&fodata)
		if err != nil {
			return rb
		}
		connPath := make([]uint8, fodata.ConnPathSize*2)
		rb, err = r.read(&connPath)
		if err != nil {
			return rb
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
		r.maxFO = int(fodata.TOConnPar&0x1FF) - 32

		r.write(r.resp)
		r.write(sr)

	case r.class == ConnManager && r.protd.Service == LargeForwOpen:
		r.p.debug("LargeForwardOpen")

		var (
			fodata largeForwardOpenData
			sr     forwardOpenResponse
		)

		rb, err := r.read(&fodata)
		if err != nil {
			return rb
		}
		connPath := make([]uint8, fodata.ConnPathSize*2)
		rb, err = r.read(&connPath)
		if err != nil {
			return rb
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
		r.maxFO = int(fodata.TOConnPar&0xFFFF) - 32

		r.write(r.resp)
		r.write(sr)

	case r.class == ConnManager && r.protd.Service == ForwardClose:
		r.p.debug("ForwardClose")

		var (
			fcdata forwardCloseData
			sr     forwardCloseResponse
		)

		rb, err := r.read(&fcdata)
		if err != nil {
			return rb
		}
		connPath := make([]uint8, fcdata.ConnPathSize*2)
		rb, err = r.read(&connPath)
		if err != nil {
			return rb
		}

		sr.ConnSerialNumber = fcdata.ConnSerialNumber
		sr.VendorID = fcdata.VendorID
		sr.OriginatorSerialNumber = fcdata.OriginatorSerialNumber
		sr.AppReplySize = 0

		r.connID = 0

		r.write(r.resp)
		r.write(sr)

	case r.class == TemplateClass && r.protd.Service == ReadTemplate:
		r.p.debug("ReadTemplate")

		var rd readTemplateResponse

		rb, err := r.read(&rd)
		if err != nil {
			return rb
		}
		r.p.debug(rd.Offset, rd.Number)

		if in := r.p.GetClassInstance(r.class, r.instance); in != nil && rd.Offset < uint32(len(in.data)) {
			data := in.data[rd.Offset:]
			if len(data) > r.maxData {
				r.resp.Status = PartialTransfer
				data = data[:r.maxData]
			}
			r.write(r.resp)
			r.write(data)
		} else {
			r.err(PathUnknown)
		}

	case r.class == 0xAC && r.protd.Service == ReadTag:
		fmt.Println("unknown service:", r.protd.Service)

		data := make([]uint8, r.dataLen)
		rb, err := r.read(&data)
		if err != nil {
			return rb
		}

		r.err(ServNotSup)

	case (r.class == -1 || r.class == SymbolClass) && r.protd.Service == ReadTag:
		r.p.debug("ReadTag")

		var tagCount uint16

		rb, err := r.read(&tagCount)
		if err != nil {
			return rb
		}

		if rtData, tagType, elLen, ok := r.p.readTag(r.path, tagCount); ok {
			if len(rtData) > r.maxData {
				r.resp.Status = PartialTransfer
				if elLen > r.maxData {
					rtData = rtData[:r.maxData]
				} else {
					rtData = rtData[:(r.maxData/elLen)*elLen]
				}
			}
			r.write(r.resp)
			if tagType >= TypeStructHead {
				r.write(uint16(tagType >> 16))
			}
			r.write(uint16(tagType))
			r.write(rtData)
		} else {
			r.resp.Status = PathSegmentError
			r.resp.AddStatusSize = 1

			r.write(r.resp)
			r.write(uint16(0))
		}

	case (r.class == -1 || r.class == SymbolClass) && r.protd.Service == ReadTagFrag:
		r.p.debug("ReadTagFragmented")

		var (
			tagCount  uint16
			tagOffset uint32
		)

		rb, err := r.read(&tagCount)
		if err != nil {
			return rb
		}
		rb, err = r.read(&tagOffset)
		if err != nil {
			return rb
		}

		if rtData, tagType, elLen, ok := r.p.readTag(r.path, tagCount); ok && tagOffset < uint32(len(rtData)) {
			rtData = rtData[tagOffset:]
			if len(rtData) > r.maxData {
				r.resp.Status = PartialTransfer
				if elLen > r.maxData {
					rtData = rtData[:r.maxData]
				} else {
					rtData = rtData[:(r.maxData/elLen)*elLen]
				}
			}
			r.write(r.resp)
			if tagType >= TypeStructHead {
				r.write(uint16(tagType >> 16))
			}
			r.write(uint16(tagType))
			r.write(rtData)
		} else {
			r.resp.Status = PathSegmentError
			r.resp.AddStatusSize = 1

			r.write(r.resp)
			r.write(uint16(0))
		}

	case (r.class == -1 || r.class == SymbolClass) && r.protd.Service == ReadModifyWrite:
		r.p.debug("ReadModifyWrite")

		var maskSize uint16

		rb, err := r.read(&maskSize)
		if err != nil {
			return rb
		}
		orMask := make([]uint8, maskSize)
		rb, err = r.read(&orMask)
		if err != nil {
			return rb
		}
		andMask := make([]uint8, maskSize)
		rb, err = r.read(&andMask)
		if err != nil {
			return rb
		}
		if r.p.readModWriteTag(r.path, orMask, andMask) {
			r.write(r.resp)
		} else {
			r.resp.Status = PathSegmentError
			r.resp.AddStatusSize = 1

			r.write(r.resp)
			r.write(uint16(0))
		}

	case (r.class == -1 || r.class == SymbolClass) && r.protd.Service == WriteTag:
		r.p.debug("WriteTag")

		var (
			tagType  uint16
			tagCount uint16
		)

		rb, err := r.read(&tagType)
		if err != nil {
			return rb
		}
		if tagType == 0x02A0 {
			rb, err = r.read(&tagType)
			if err != nil {
				return rb
			}
			r.dataLen -= 2
		}
		rb, err = r.read(&tagCount)
		if err != nil {
			return rb
		}

		wrData := make([]uint8, r.dataLen-4)
		rb, err = r.read(wrData)
		if err != nil {
			return rb
		}

		if r.p.saveTag(r.path, tagType, int(tagCount), wrData, 0) {
			r.write(r.resp)
		} else {
			r.resp.Status = PathSegmentError
			r.resp.AddStatusSize = 1

			r.write(r.resp)
			r.write(uint16(0))
		}

	case (r.class == -1 || r.class == SymbolClass) && r.protd.Service == WriteTagFrag:
		r.p.debug("WriteTagFragmented")

		var (
			tagType   uint16
			tagCount  uint16
			tagOffset uint32
		)

		rb, err := r.read(&tagType)
		if err != nil {
			return rb
		}
		if tagType == 0x02A0 {
			rb, err = r.read(&tagType)
			if err != nil {
				return rb
			}
			r.dataLen -= 2
		}
		rb, err = r.read(&tagCount)
		if err != nil {
			return rb
		}
		rb, err = r.read(&tagOffset)
		if err != nil {
			return rb
		}

		wrData := make([]uint8, r.dataLen-8)
		rb, err = r.read(wrData)
		if err != nil {
			return rb
		}

		if r.p.saveTag(r.path, tagType, (r.dataLen-8)/int(typeLen(tagType)), wrData, int(tagOffset)) {
			r.write(r.resp)
		} else {
			r.resp.Status = PathSegmentError
			r.resp.AddStatusSize = 1

			r.write(r.resp)
			r.write(uint16(0))
		}

	case r.protd.Service == Reset:
		r.p.debug("Reset")

		data := make([]uint8, r.dataLen)
		rb, err := r.read(&data)
		if err != nil {
			return rb
		}

		if r.dataLen >= 1 && data[0] > 1 {
			r.resp.Status = InvalidPar
		}

		if r.p.callback != nil {
			go r.p.callback(Reset, int(r.resp.Status), nil)
		}

		r.write(r.resp)

	case r.protd.Service == NextInst:
		r.p.debug("FindNextObjectInstance")
		var (
			count uint8
			buf   bytes.Buffer
		)

		rb, err := r.read(&count)
		if err != nil {
			return rb
		}

		li, _ := r.p.GetClassInstancesList(r.class, r.instance, int(count))
		if li != nil {
			bwrite(&buf, uint8(len(li)))
			for _, x := range li {
				bwrite(&buf, uint16(x))
			}

			r.write(r.resp)
			r.write(buf.Bytes())
		} else {
			r.err(PathUnknown)
		}

	case r.protd.Service == GetMember:
		r.p.debug("GetMember")

		fmt.Println(r.class, r.instance, r.attr, r.member)

		at, aok, in := r.p.GetClassInstanceAttr(r.class, r.instance, r.attr)
		r.resp.Service = r.protd.Service + 128

		if in && aok && at.st != nil && at.st.l > 0 {
			r.p.debug(at.Name)
			from := r.member * at.st.l
			to := from + at.st.l
			if to > len(at.data) {
				return r.err(InvalidPar)
			}
			r.write(r.resp)
			r.write(at.data[from:to])
		} else {
			r.err(ServNotSup)
		}

	default:
		fmt.Println("unknown service:", r.protd.Service)

		data := make([]uint8, r.dataLen)
		rb, err := r.read(&data)
		if err != nil {
			return rb
		}

		r.err(ServNotSup)
	}
	return true
}
