// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package plcconnector implements communication with PLC.
package plcconnector

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Service
const (
	Reset        = 0x05
	ForwardOpen  = 0x54
	ForwardClose = 0x4e
	ReadTag      = 0x4c
	WriteTag     = 0x4d
)

// Data types
const (
	TypeBOOL  = 0xc1 // 1 byte
	TypeSINT  = 0xc2 // 1 byte
	TypeINT   = 0xc3 // 2 bytes
	TypeDINT  = 0xc4 // 4 bytes
	TypeREAL  = 0xca // 4 bytes
	TypeDWORD = 0xd3 // 4 bytes
	TypeLINT  = 0xc5 // 8 bytes
)

// Status codes
const (
	Success          = 0x00
	PathSegmentError = 0x04
)

const (
	registerSession   = 0x65
	sendRRData        = 0x6f
	sendUnitData      = 0x70
	unregisterSession = 0x66

	nullAddressItem = 0x00
	unconnDataItem  = 0xb2
	connAddressItem = 0xa1
	connDataItem    = 0xb1

	ansiExtended = 0x91
)

const (
	timeout = 60
)

type encapsulationHeader struct {
	Command       uint16
	Length        uint16
	SessionHandle uint32
	Status        uint32
	SenderContext uint64
	Options       uint32
}

type registerSessionData struct {
	ProtocolVersion uint16
	OptionFlags     uint16
}

type sendData struct {
	InterfaceHandle uint32
	Timeout         uint16
	ItemCount       uint16
}

type itemType struct {
	Type   uint16
	Length uint16
}

type protocolData struct {
	Service  uint8
	PathSize uint8
}

type forwardOpenData struct {
	TimeOut                uint16
	OTConnectionID         uint32
	TOConnectionID         uint32
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	ConnTimeoutMult        uint8
	_                      [3]uint8
	OTRPI                  uint32
	OTConnPar              uint16
	TORPI                  uint32
	TOConnPar              uint16
	TransportType          uint8
	ConnPathSize           uint8
	ConnPath               [6]uint8
}

type forwardCloseData struct {
	TimeOut                uint16
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	ConnPathSize           uint8
	_                      uint8
	ConnPath               [6]uint8
}

type forwardOpenResponse struct {
	Service                uint8
	_                      uint8
	Status                 uint16
	OTConnectionID         uint32
	TOConnectionID         uint32
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	OTAPI                  uint32
	TOAPI                  uint32
	AppReplySize           uint8
	_                      uint8
}

type forwardCloseResponse struct {
	Service                uint8
	_                      uint8
	Status                 uint16
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	AppReplySize           uint8
	_                      uint8
}

type readTagResponse struct {
	Service uint8
	_       uint8
	Status  uint16
	TagType uint16
}

type response struct {
	Service uint8
	_       uint8
	Status  uint16
}

type errorResponse struct {
	Service       uint8
	_             uint8
	Status        uint8
	AddStatusSize uint8
	AddStatus     uint16
}

// Tag .
type Tag struct {
	Name  string
	Typ   int
	Count int

	data []uint8
}

// DataBytes returns array of bytes.
func (t *Tag) DataBytes() []byte {
	return t.data
}

// DataBOOL returns array of BOOL.
func (t *Tag) DataBOOL() []bool {
	ret := make([]bool, 0, t.Count)
	for i := 0; i < len(t.data); i++ {
		tmp := false
		if t.data[i] != 0 {
			tmp = true
		}
		ret = append(ret, tmp)
	}
	return ret
}

// DataSINT returns array of int8.
func (t *Tag) DataSINT() []int8 {
	ret := make([]int8, 0, t.Count)
	for i := 0; i < len(t.data); i++ {
		ret = append(ret, int8(t.data[i]))
	}
	return ret
}

// DataINT returns array of int16.
func (t *Tag) DataINT() []int16 {
	ret := make([]int16, 0, t.Count)
	for i := 0; i < len(t.data); i += 2 {
		tmp := int16(t.data[i])
		tmp += int16(t.data[i+1]) << 8
		ret = append(ret, tmp)
	}
	return ret
}

// DataDINT returns array of int32.
func (t *Tag) DataDINT() []int32 {
	ret := make([]int32, 0, t.Count)
	for i := 0; i < len(t.data); i += 4 {
		tmp := int32(t.data[i])
		tmp += int32(t.data[i+1]) << 8
		tmp += int32(t.data[i+2]) << 16
		tmp += int32(t.data[i+3]) << 24
		ret = append(ret, tmp)
	}
	return ret
}

// DataREAL returns array of float32.
func (t *Tag) DataREAL() []float32 {
	ret := make([]float32, 0, t.Count)
	for i := 0; i < len(t.data); i += 4 {
		tmp := uint32(t.data[i])
		tmp += uint32(t.data[i+1]) << 8
		tmp += uint32(t.data[i+2]) << 16
		tmp += uint32(t.data[i+3]) << 24
		ret = append(ret, math.Float32frombits(tmp))
	}
	return ret
}

// DataDWORD returns array of int32.
func (t *Tag) DataDWORD() []int32 {
	return t.DataDINT()
}

// DataLINT returns array of int64.
func (t *Tag) DataLINT() []int64 {
	ret := make([]int64, 0, t.Count)
	for i := 0; i < len(t.data); i += 8 {
		tmp := int64(t.data[i])
		tmp += int64(t.data[i+1]) << 8
		tmp += int64(t.data[i+2]) << 16
		tmp += int64(t.data[i+3]) << 24
		tmp += int64(t.data[i+4]) << 32
		tmp += int64(t.data[i+5]) << 40
		tmp += int64(t.data[i+6]) << 48
		tmp += int64(t.data[i+7]) << 56
		ret = append(ret, tmp)
	}
	return ret
}

var (
	tags    map[string]*Tag
	tMut    sync.RWMutex
	verbose = false
)

func typeLen(t uint16) uint16 {
	switch t {
	case TypeBOOL:
		return 1
	case TypeSINT:
		return 1
	case TypeINT:
		return 2
	case TypeDINT:
		return 4
	case TypeREAL:
		return 4
	case TypeDWORD:
		return 4
	case TypeLINT:
		return 8
	}
	return 1
}

func debug(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func readData(r io.Reader, data interface{}) error {
	err := binary.Read(r, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	if verbose {
		fmt.Printf("%#v\n", data)
	}
	return err
}

func writeData(w io.Writer, data interface{}) {
	err := binary.Write(w, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func readTag(tag string, count uint16) ([]uint8, uint16, bool) {
	tMut.RLock()
	defer tMut.RUnlock()
	t, ok := tags[tag]
	debug(t, ok)
	if ok && count <= uint16(t.Count) {
		if callback != nil {
			callback(ReadTag, Success, t)
		}
		return t.data, uint16(t.Typ), true
	}
	if callback != nil {
		callback(ReadTag, PathSegmentError, nil)
	}
	return nil, 0, false
}

func saveTag(tag string, typ, count uint16, data []uint8) bool {
	t := Tag{Name: tag, Typ: int(typ), Count: int(count), data: data}
	tMut.Lock()
	tags[tag] = &t
	if callback != nil {
		callback(WriteTag, Success, &t)
	}
	tMut.Unlock()
	return true
}

// Init initialize library. Must be called first.
func Init() {
	tags = make(map[string]*Tag)

	tags["testBOOL"] = &Tag{Name: "testBOOL", Typ: TypeBOOL, Count: 4, data: []uint8{
		0x00, 0x01, 0xFF, 0x55}}

	tags["testSINT"] = &Tag{Name: "testSINT", Typ: TypeSINT, Count: 4, data: []uint8{
		0xFF, 0xFE, 0x00, 0x01}}

	tags["testINT"] = &Tag{Name: "testINT", Typ: TypeINT, Count: 2, data: []uint8{
		0xFF, 0xFF, 0x00, 0x01}}

	tags["testDINT"] = &Tag{Name: "testDINT", Typ: TypeDINT, Count: 2, data: []uint8{
		0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x00, 0x00, 0x00}}

	tags["testREAL"] = &Tag{Name: "testREAL", Typ: TypeREAL, Count: 2, data: []uint8{
		0xFE, 0xFE, 0x00, 0x00,
		0xFE, 0x00, 0x00, 0x00}}

	tags["testDWORD"] = &Tag{Name: "testDWORD", Typ: TypeDWORD, Count: 2, data: []uint8{
		0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x00, 0x00, 0x00}}

	tags["testLINT"] = &Tag{Name: "testLINT", Typ: TypeLINT, Count: 2, data: []uint8{
		0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}}
}

// AddTag adds tag.
func AddTag(t Tag) {
	size := typeLen(uint16(t.Typ)) * uint16(t.Count)
	t.data = make([]uint8, size)
	tMut.Lock()
	tags[t.Name] = &t
	tMut.Unlock()
}

// UpdateTag sets data to the tag
func UpdateTag(name string, offset int, data []uint8) bool {
	tMut.Lock()
	defer tMut.Unlock()
	t, ok := tags[name]
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

var callback func(service int, statut int, tag *Tag)

// Callback registers function called at receiving communication with PLC.
// tag may be nil in event of error or reset.
func Callback(function func(service int, status int, tag *Tag)) {
	callback = function
}

// SetVerbose enables debugging output.
func SetVerbose(on bool) {
	verbose = on
}

var (
	closeWait *sync.Cond
	closeWMut sync.Mutex
	closeMut  sync.RWMutex
	closeI    bool
)

// Serve listens on the TCP network address host.
func Serve(host string) error {
	rand.Seed(time.Now().UnixNano())

	closeWait = sync.NewCond(&closeWMut)

	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return err
	}
	serv, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	for {
		serv.SetDeadline(time.Now().Add(time.Second))
		conn, err := serv.AcceptTCP()
		if e, ok := err.(net.Error); ok && e.Timeout() {
			closeMut.RLock()
			endP := closeI
			closeMut.RUnlock()
			if endP {
				break
			}
		} else if err != nil {
			fmt.Println("plcconnector Serve: ", err)
			return err
		} else {
			go handleRequest(conn)
		}
	}
	debug("Serve shutdown")
	closeWait.Signal()
	return nil
}

// Close shutdowns server
func Close() {
	closeMut.Lock()
	closeI = true
	closeMut.Unlock()
	closeWait.L.Lock()
	closeWait.Wait()
	closeWait.L.Unlock()
}

func handleRequest(conn net.Conn) {
	connID := uint32(0)

	readBuf := bufio.NewReader(conn)
	writeBuf := new(bytes.Buffer)

loop:
	for {
		readBuf.Reset(conn)
		writeBuf.Reset()

		closeMut.RLock()
		endP := closeI
		closeMut.RUnlock()
		if endP {
			break loop
		}

		err := conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
		if err != nil {
			fmt.Println(err)
			break loop
		}

		debug()
		var encHead encapsulationHeader
		err = readData(readBuf, &encHead)
		if err != nil {
			break loop
		}

		switch encHead.Command {
		case registerSession:
			debug("RegisterSession")

			var data registerSessionData
			err = readData(readBuf, &data)
			if err != nil {
				break loop
			}

			encHead.SessionHandle = rand.Uint32()

			writeData(writeBuf, encHead)
			writeData(writeBuf, data)

		case unregisterSession:
			debug("UnregisterSession")
			break loop

		case sendRRData, sendUnitData:
			debug("SendRRData/SendUnitData")

			var (
				data         sendData
				item         itemType
				dataLen      uint16
				addrLen      uint16
				protd        protocolData
				protSeqCount uint16
			)
			err = readData(readBuf, &data)
			if err != nil {
				break loop
			}

			data.Timeout = 0

			for i := uint16(0); i < data.ItemCount; i++ {
				err = readData(readBuf, &item)
				if err != nil {
					break loop
				}
				if item.Length > 0 && item.Type != unconnDataItem && item.Type != connDataItem {
					itemdata := make([]uint8, item.Length)
					err = readData(readBuf, &itemdata)
					if err != nil {
						break loop
					}
				}
				if item.Type == connDataItem {
					err = readData(readBuf, &protSeqCount)
					if err != nil {
						break loop
					}
				}
			}

			err = readData(readBuf, &protd)
			if err != nil {
				break loop
			}

			protdPath := make([]uint8, protd.PathSize*2)
			err = readData(readBuf, &protdPath)
			if err != nil {
				break loop
			}

			switch protd.Service {
			case ForwardOpen:
				debug("ForwardOpen")

				var (
					fodata forwardOpenData
					resp   forwardOpenResponse
				)
				err = readData(readBuf, &fodata)
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
				writeData(writeBuf, encHead)
				writeData(writeBuf, data)
				writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				writeData(writeBuf, resp)

			case ForwardClose:
				debug("ForwardClose")

				var (
					fcdata forwardCloseData
					resp   forwardCloseResponse
				)
				err = readData(readBuf, &fcdata)
				if err != nil {
					break loop
				}

				resp.Service = ForwardClose | 128
				resp.Status = 0
				resp.ConnSerialNumber = fcdata.ConnSerialNumber
				resp.VendorID = fcdata.VendorID
				resp.OriginatorSerialNumber = fcdata.OriginatorSerialNumber
				resp.AppReplySize = 0

				encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
				writeData(writeBuf, encHead)
				writeData(writeBuf, data)
				writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				writeData(writeBuf, resp)

			case ReadTag:
				debug("ReadTag")

				var (
					tagName  string
					tagCount uint16
				)

				if protd.PathSize > 0 && protdPath[0] == ansiExtended {
					tagName = string(protdPath[2 : protdPath[1]+2])
				}
				err = readData(readBuf, &tagCount)
				if err != nil {
					break loop
				}
				debug(tagName, tagCount)

				if rtData, rtType, ok := readTag(tagName, tagCount); ok {
					var resp readTagResponse

					resp.Service = ReadTag | 128
					resp.Status = Success
					resp.TagType = rtType

					dataLen = uint16(binary.Size(resp)) + typeLen(resp.TagType)*tagCount
					addrLen = 0

					if connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					writeData(writeBuf, encHead)
					writeData(writeBuf, data)
					if connID != 0 {
						writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						writeData(writeBuf, connID)
						writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						writeData(writeBuf, protSeqCount)
					} else {
						writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					writeData(writeBuf, resp)
					writeData(writeBuf, rtData)

				} else {
					var resp errorResponse

					resp.Service = ReadTag | 128
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1
					resp.AddStatus = 0

					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					writeData(writeBuf, encHead)
					writeData(writeBuf, data)
					if connID != 0 {
						writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						writeData(writeBuf, connID)
						writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						writeData(writeBuf, protSeqCount)
					} else {
						writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					writeData(writeBuf, resp)
				}

			case WriteTag:
				debug("WriteTag")

				var (
					tagName  string
					tagType  uint16
					tagCount uint16
				)

				if protd.PathSize > 0 && protdPath[0] == ansiExtended {
					tagName = string(protdPath[2 : protdPath[1]+2])
				}
				err = readData(readBuf, &tagType)
				if err != nil {
					break loop
				}
				err = readData(readBuf, &tagCount)
				if err != nil {
					break loop
				}
				debug(tagName, tagType, tagCount)

				wrData := make([]uint8, typeLen(tagType)*tagCount)
				err = readData(readBuf, wrData)
				if err != nil {
					break loop
				}

				if saveTag(tagName, tagType, tagCount, wrData) {
					var resp response

					resp.Service = WriteTag | 128
					resp.Status = Success

					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					writeData(writeBuf, encHead)
					writeData(writeBuf, data)
					if connID != 0 {
						writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						writeData(writeBuf, connID)
						writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						writeData(writeBuf, protSeqCount)
					} else {
						writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					writeData(writeBuf, resp)
				} else {
					var resp errorResponse

					resp.Service = WriteTag | 128
					resp.Status = PathSegmentError
					resp.AddStatusSize = 1
					resp.AddStatus = 0

					dataLen = uint16(binary.Size(resp))
					addrLen = 0

					if connID != 0 {
						dataLen += uint16(binary.Size(protSeqCount))
						addrLen = uint16(binary.Size(connID))
					}

					encHead.Length = uint16(binary.Size(data)+2*binary.Size(itemType{})) + addrLen + dataLen
					writeData(writeBuf, encHead)
					writeData(writeBuf, data)
					if connID != 0 {
						writeData(writeBuf, itemType{Type: connAddressItem, Length: addrLen})
						writeData(writeBuf, connID)
						writeData(writeBuf, itemType{Type: connDataItem, Length: dataLen})
						writeData(writeBuf, protSeqCount)
					} else {
						writeData(writeBuf, itemType{Type: nullAddressItem, Length: addrLen})
						writeData(writeBuf, itemType{Type: unconnDataItem, Length: dataLen})
					}
					writeData(writeBuf, resp)
				}

			case Reset:
				debug("Reset")

				var resp response

				resp.Service = Reset + 128

				if callback != nil {
					callback(Reset, Success, nil)
				}

				encHead.Length = uint16(binary.Size(data) + 2*binary.Size(itemType{}) + binary.Size(resp))
				writeData(writeBuf, encHead)
				writeData(writeBuf, data)
				writeData(writeBuf, itemType{Type: nullAddressItem, Length: 0})
				writeData(writeBuf, itemType{Type: unconnDataItem, Length: uint16(binary.Size(resp))})
				writeData(writeBuf, resp)

			default:
				writeData(writeBuf, encHead)
				writeData(writeBuf, data)
				writeData(writeBuf, protd)
			}

		default:
			debug(encHead.Command)

			data := make([]uint8, encHead.Length)
			err = readData(readBuf, &data)
			if err != nil {
				break loop
			}

			writeData(writeBuf, encHead)
			writeData(writeBuf, data)
		}

		err = conn.SetWriteDeadline(time.Now().Add(timeout * time.Second))
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
