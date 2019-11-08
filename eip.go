package plcconnector

import (
	"encoding/binary"
	"errors"
	"math/rand"
)

const (
	ansiExtended = 0x91

	pathType    = 0xE0
	pathLogical = 0x20

	pathSegType   = 0x1C
	pathClass     = 0x00
	pathInstance  = 0x04
	pathElement   = 0x08
	pathAttribute = 0x10

	pathSize = 0x03
	path8    = 0x00
	path16   = 0x01
	path32   = 0x02
)

var errPath = errors.New("path error")

type pathEl struct {
	typ int
	val int
	txt string
}

func (r *req) parsePath(path []uint8) (int, int, int, []pathEl, error) {
	class := -1
	insta := -1
	attri := -1
	pth := []pathEl{}
	x := 0
	for i := 0; i < len(path); i++ {
		if path[i] == ansiExtended {
			ln := path[i+1]
			ansi := string(path[i+2 : i+int(ln)+2])
			i += int(ln) + 1
			if ln&1 == 1 {
				i++
			}
			pth = append(pth, pathEl{typ: ansiExtended, txt: ansi})
		} else if (path[i] & pathType) == pathLogical {
			typ := path[i] & pathSegType
			size := path[i] & pathSize
			el := 0
			switch size {
			case path8:
				el = int(path[i+1])
				i++
			case path16:
				el = int(path[i+2]) + (int(path[i+3]) << 8)
				i += 3
			case path32:
				el = int(path[i+2]) + (int(path[i+3]) << 8) + (int(path[i+4]) << 16) + (int(path[i+5]) << 24)
				i += 5
			default:
				r.p.debug("path size error")
				return 0, 0, 0, nil, errPath
			}
			switch typ {
			case pathClass:
				if class == -1 && x == 0 {
					class = el
				}
				pth = append(pth, pathEl{typ: pathClass, val: el})
			case pathInstance:
				if insta == -1 && x == 1 {
					insta = el
				}
				pth = append(pth, pathEl{typ: pathInstance, val: el})
			case pathAttribute:
				if attri == -1 && x == 2 {
					attri = el
				}
				pth = append(pth, pathEl{typ: pathAttribute, val: el})
			case pathElement:
				if attri == -1 && x == 2 {
					attri = el
				}
				pth = append(pth, pathEl{typ: pathElement, val: el})
			default:
				r.p.debug("path segment type error")
				return 0, 0, 0, nil, errPath
			}
		} else {
			r.p.debug("path type error", path[i])
		}
		x++
	}
	return class, insta, attri, pth, nil
}

// ParsePathT .
func ParsePathT() {
	var r req
	r.p = &PLC{Verbose: true}
	r.parsePath([]uint8{0x91, 0x05, 0x70, 0x61, 0x72, 0x74, 0x73, 0x00})
	r.parsePath([]uint8{0x20, 0x6B, 0x25, 0x00, 0x82, 0x25})
	r.parsePath([]uint8{0x91, 0x09, 0x73, 0x65, 0x74, 0x70, 0x6F, 0x69, 0x6E, 0x74, 0x73, 0x00, 0x28, 0x05})
	r.parsePath([]uint8{0x91, 0x07, 0x70, 0x72, 0x6f, 0x66, 0x69, 0x6c, 0x65, 0x00, 0x28, 0x00, 0x28, 0x01, 0x29, 0x00, 0x01, 0x01})
	r.parsePath([]uint8{0x20, 0x6B, 0x25, 0x00, 0x97, 0x8A, 0x28, 0x00, 0x28, 0x01, 0x29, 0x00, 0x01, 0x01})
	r.parsePath([]uint8{0x91, 0x06, 0x64, 0x77, 0x65, 0x6C, 0x6C, 0x33, 0x91, 0x03, 0x61, 0x63, 0x63, 0x00})
	r.parsePath([]uint8{0x91, 0x0A, 0x45, 0x72, 0x72, 0x6F, 0x72, 0x4C, 0x69, 0x6D, 0x69, 0x74, 0x91, 0x03, 0x50, 0x52, 0x45, 0x00})
	r.parsePath([]uint8{0x20, 0x6B, 0x25, 0x00, 0xD1, 0x18, 0x91, 0x05, 0x74, 0x6F, 0x64, 0x61, 0x79, 0x00, 0x91, 0x04, 0x72, 0x61, 0x74, 0x65})
	r.parsePath([]uint8{0x20, 0x6B, 0x25, 0x00, 0x4B, 0x0D, 0x28, 0x00, 0x91, 0x07, 0x6D, 0x79, 0x61, 0x72, 0x72, 0x61, 0x79, 0x00, 0x28, 0x01, 0x91, 0x05, 0x74, 0x6F, 0x64, 0x61, 0x79, 0x00, 0x91, 0x0B, 0x68, 0x6F, 0x75, 0x72, 0x6C, 0x79, 0x43, 0x6F, 0x75, 0x6E, 0x74, 0x00, 0x28, 0x03})
	r.parsePath([]uint8{0x91, 0x07, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x32, 0x00, 0x91, 0x08, 0x70, 0x69, 0x6C, 0x6F, 0x74, 0x5F, 0x6F, 0x6E})
}

func (r *req) eipNOP() error {
	r.p.debug("NOP")

	data := make([]byte, r.encHead.Length)
	err := r.read(&data)
	if err != nil {
		return err
	}
	return nil
}

func (r *req) eipRegisterSession() error {
	r.p.debug("RegisterSession")

	var data registerSessionData
	err := r.read(&data)
	if err != nil {
		return err
	}

	if data.ProtocolVersion > 1 {
		r.encHead.Status = eipInvalidProtocolVersion
		data.ProtocolVersion = 1
	} else {
		r.encHead.SessionHandle = rand.Uint32()
	}

	r.write(data)
	return nil
}

func (r *req) eipListIdentity() error {
	var (
		data listIdentityData
		typ  itemType
	)

	data.ProtocolVersion = 1
	data.SocketFamily = htons(2)
	data.SocketPort = htons(r.p.port)
	data.SocketAddr = getIP4()

	attrs := r.p.Class[0x01].inst[0x01].getAttrList([]int{1, 2, 3, 4, 5, 6, 7, 8})

	typ.Type = itListIdentity
	typ.Length = uint16(binary.Size(data) + len(attrs))

	r.write(uint16(1)) // ItemCount
	r.write(typ)
	r.write(data)
	r.write(attrs)
	return nil
}

func (r *req) eipListServices() error {
	var (
		data listServicesData
		typ  itemType
	)

	typ.Type = itListService
	typ.Length = uint16(binary.Size(data))

	data.ProtocolVersion = 1
	data.CapabilityFlags = lscfTCP
	data.NameOfService = [16]int8{67, 111, 109, 109, 117, 110, 105, 99, 97, 116, 105, 111, 110, 115, 0, 0} // Communications

	r.write(uint16(1)) // ItemCount
	r.write(typ)
	r.write(data)
	return nil
}
