package plcconnector

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"strconv"
	"unicode"
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

func parsePath(p string) []pathEl {
	e := make([]pathEl, 0, 4)

	name := ""
	no := ""
	inName := true
	last := rune(0)

	for _, t := range p {
		switch {
		case t == '.':
			if name == "" {
				if last != ']' {
					return nil
				}
				break
			}
			e = append(e, pathEl{typ: ansiExtended, txt: name})
			name = ""
			inName = true
		case t == '[':
			if name == "" {
				return nil
			}
			inName = false
			e = append(e, pathEl{typ: ansiExtended, txt: name})
			name = ""
		case t == ']':
			if no == "" {
				return nil
			}
			inName = true
			noint, _ := strconv.Atoi(no)
			e = append(e, pathEl{typ: pathElement, val: noint})
			no = ""
		case unicode.IsDigit(t):
			if inName {
				if name == "" {
					return nil
				}
				name += string(t)
			} else {
				no += string(t)
			}
		case unicode.IsLetter(t) || t == ':' || t == '_':
			if !inName {
				return nil
			}
			name += string(t)
		default:
			return nil
		}
		last = t
	}
	if last == '.' || last == '[' {
		return nil
	}
	if name != "" {
		e = append(e, pathEl{typ: ansiExtended, txt: name})
	}

	return e
}

func constructPath(p []pathEl) []uint8 {
	if p == nil {
		return nil
	}
	b := make([]uint8, 0, len(p)*10)
	for _, e := range p {
		switch e.typ {
		case ansiExtended:
			byt := []uint8(e.txt)
			if len(byt) > 255 {
				return nil
			}
			b = append(b, []uint8{ansiExtended, uint8(len(byt))}...)
			b = append(b, byt...)
			if len(byt)&1 == 1 {
				b = append(b, 0)
			}
		case pathElement: // 32 bit?
			if e.val > 65535 {
				return nil
			} else if e.val > 255 {
				b = append(b, []uint8{pathLogical | pathElement | path16, 0, uint8(e.val), uint8(e.val >> 8)}...)
			} else {
				b = append(b, []uint8{pathLogical | pathElement, uint8(e.val)}...)
			}
		default:
			return nil
		}
	}
	return b
}

func (r *req) parsePath(path []uint8) (int, int, int, []pathEl, error) {
	class := -1
	insta := -1
	attri := -1
	pth := []pathEl{}
	x := 0
	for i := 0; i < len(path); i++ {
		if path[i] == ansiExtended && i+1 < len(path) && i+1+int(path[i+1]) < len(path) {
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
			switch {
			case size == path8 && i+1 < len(path):
				el = int(path[i+1])
				i++
			case size == path16 && i+3 < len(path):
				el = int(path[i+2]) + (int(path[i+3]) << 8)
				i += 3
			case size == path32 && i+5 < len(path):
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
			return 0, 0, 0, nil, errPath
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
	data.SocketAddr, _ = getNetIf()

	attrs := r.p.Class[IdentityClass].inst[0x01].getAttrList([]int{1, 2, 3, 4, 5, 6, 7, 8})

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
