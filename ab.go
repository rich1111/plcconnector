// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package plcconnector .
package plcconnector

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	Typ   uint16
	Count uint16
	Data  []uint8
}

var (
	tags    map[string]Tag
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
	t, ok := tags[tag]
	tMut.RUnlock()
	fmt.Println(t, ok)
	if ok && count <= t.Count {
		if callback != nil {
			callback(ReadTag, Success, &t)
		}
		return t.Data, t.Typ, true
	}
	if callback != nil {
		callback(ReadTag, PathSegmentError, nil)
	}
	return nil, 0, false
}

func saveTag(tag string, typ, count uint16, data []uint8) bool {
	t := Tag{Typ: typ, Count: count, Data: data}
	tMut.Lock()
	tags[tag] = t
	tMut.Unlock()
	if callback != nil {
		callback(WriteTag, Success, &t)
	}
	return true
}

// Init initialize library. Must be called first.
func Init() {
	tags = make(map[string]Tag)

	tags["testBOOL"] = Tag{Typ: TypeBOOL, Count: 4, Data: []uint8{
		0x00, 0x01, 0xFF, 0x55}}

	tags["testSINT"] = Tag{Typ: TypeSINT, Count: 4, Data: []uint8{
		0xFF, 0xFE, 0x00, 0x01}}

	tags["testINT"] = Tag{Typ: TypeINT, Count: 2, Data: []uint8{
		0xFF, 0xFF, 0x00, 0x01}}

	tags["testDINT"] = Tag{Typ: TypeDINT, Count: 2, Data: []uint8{
		0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x00, 0x00, 0x00}}

	tags["testREAL"] = Tag{Typ: TypeREAL, Count: 2, Data: []uint8{
		0xFE, 0xFE, 0x00, 0x00,
		0xFE, 0x00, 0x00, 0x00}}

	tags["testDWORD"] = Tag{Typ: TypeDWORD, Count: 2, Data: []uint8{
		0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x00, 0x00, 0x00}}

	tags["testLINT"] = Tag{Typ: TypeLINT, Count: 2, Data: []uint8{
		0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}}
}

var callback func(service int, statut int, tag *Tag)

// Callback .
func Callback(function func(service int, status int, tag *Tag)) {
	callback = function
}

// SetVerbose enables debugging output.
func SetVerbose(on bool) {
	verbose = on
}

// Serve listens on the TCP network address host.
func Serve(host string) error {
	rand.Seed(time.Now().UnixNano())

	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return err
	}
	serv, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := serv.AcceptTCP()
		if err != nil {
			return err
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	connID := uint32(0)

	readBuf := bufio.NewReader(conn)
	writeBuf := new(bytes.Buffer)

loop:
	for {
		readBuf.Reset(conn)
		writeBuf.Reset()

		err := conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
		if err != nil {
			fmt.Println(err)
			break loop
		}

		fmt.Println()
		var encHead encapsulationHeader
		err = readData(readBuf, &encHead)
		if err != nil {
			break loop
		}

		switch encHead.Command {
		case registerSession:
			fmt.Println("RegisterSession")

			var data registerSessionData
			err = readData(readBuf, &data)
			if err != nil {
				break loop
			}

			encHead.SessionHandle = rand.Uint32()

			writeData(writeBuf, encHead)
			writeData(writeBuf, data)

		case unregisterSession:
			fmt.Println("UnregisterSession")
			break loop

		case sendRRData, sendUnitData:
			fmt.Println("SendRRData/SendUnitData")

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
				fmt.Println("ForwardOpen")

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
				fmt.Println("ForwardClose")

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
				fmt.Println("ReadTag")

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
				fmt.Println(tagName, tagCount)

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
				fmt.Println("WriteTag")

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
				fmt.Println(tagName, tagType, tagCount)

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
				fmt.Println("Reset")

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
			fmt.Println(encHead.Command)

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
