package plcconnector

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"unicode"
)

const (
	ansiExtended = 0x91
	pathBit      = 0xFF

	pathType    = 0xE0
	pathLogical = 0x20

	pathSegType   = 0x1C
	pathClass     = 0x00
	pathInstance  = 0x04
	pathMember    = 0x08
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
	bit := ""
	inBit := false
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
			inName = false
			if last != ']' {
				if name == "" {
					return nil
				}
				e = append(e, pathEl{typ: ansiExtended, txt: name})
				name = ""
			}
		case t == ']':
			if strings.TrimSpace(no) == "" {
				return nil
			}
			inName = true
			noint, _ := strconv.Atoi(strings.TrimSpace(no))
			e = append(e, pathEl{typ: pathMember, val: noint})
			no = ""
		case t == ',':
			if inName || strings.TrimSpace(no) == "" {
				return nil
			}
			noint, _ := strconv.Atoi(strings.TrimSpace(no))
			e = append(e, pathEl{typ: pathMember, val: noint})
			no = ""
		case unicode.IsDigit(t) || t == ' ':
			if inName {
				if t == ' ' {
					return nil
				} else if name == "" && bit == "" {
					if last == '.' {
						inBit = true
						bit += string(t)
					} else {
						return nil
					}
				} else if inBit {
					bit += string(t)
				} else {
					name += string(t)
				}
			} else {
				no += string(t)
			}
		case unicode.IsLetter(t) || t == ':' || t == '_':
			if inName && inBit {
				return nil
			}
			if !inName {
				return nil
			}
			name += string(t)
		default:
			return nil
		}
		last = t
	}
	if last == '.' || last == '[' || no != "" {
		return nil
	}

	if name != "" {
		e = append(e, pathEl{typ: ansiExtended, txt: name})
	}

	if bit != "" {
		bitint, _ := strconv.Atoi(bit)
		e = append(e, pathEl{typ: pathBit, val: bitint})
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
		case pathMember:
			if e.val > 65535 {
				b = append(b, []uint8{pathLogical | pathMember | path32, 0, uint8(e.val), uint8(e.val >> 8), uint8(e.val >> 16), uint8(e.val >> 24)}...)
			} else if e.val > 255 {
				b = append(b, []uint8{pathLogical | pathMember | path16, 0, uint8(e.val), uint8(e.val >> 8)}...)
			} else {
				b = append(b, []uint8{pathLogical | pathMember, uint8(e.val)}...)
			}
		case pathBit:
		default:
			return nil
		}
	}
	return b
}

func (r *req) parsePath(path []uint8) (int, int, int, int, []pathEl, error) {
	class := -1
	insta := -1
	attri := -1
	membi := -1
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
				return 0, 0, 0, 0, nil, errPath
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
			case pathMember:
				if attri == -1 && x == 2 {
					attri = el
				} else if membi == -1 && x == 3 {
					membi = el
				}
				pth = append(pth, pathEl{typ: pathMember, val: el})
			default:
				r.p.debug("path segment type error")
				return 0, 0, 0, 0, nil, errPath
			}
		} else {
			r.p.debug("path type error", path[i])
			return 0, 0, 0, 0, nil, errPath
		}
		x++
	}
	return class, insta, attri, membi, pth, nil
}

func (r *req) eipNOP() error {
	r.p.debug("NOP")

	data := make([]byte, r.encHead.Length)
	_, err := r.read(&data)
	if err != nil {
		return err
	}
	return nil
}

func (r *req) eipRegisterSession() error {
	r.p.debug("RegisterSession")

	var data registerSessionData
	_, err := r.read(&data)
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

func pathCIA(clas, instance, attr, member int) []uint8 {
	path := make([]uint8, 0, 6)
	if clas > 65535 {
		path = append(path, []uint8{0x22, 0, uint8(clas), uint8(clas >> 8), uint8(clas >> 16), uint8(clas >> 24)}...)
	} else if clas > 255 {
		path = append(path, []uint8{0x21, 0, uint8(clas), uint8(clas >> 8)}...)
	} else if clas >= 0 {
		path = append(path, []uint8{0x20, uint8(clas)}...)
	}
	if instance > 65535 {
		path = append(path, []uint8{0x26, 0, uint8(instance), uint8(instance >> 8), uint8(instance >> 16), uint8(instance >> 24)}...)
	} else if instance > 255 {
		path = append(path, []uint8{0x25, 0, uint8(instance), uint8(instance >> 8)}...)
	} else if instance >= 0 {
		path = append(path, []uint8{0x24, uint8(instance)}...)
	}
	if attr > 65535 {
		path = append(path, []uint8{0x32, 0, uint8(attr), uint8(attr >> 8), uint8(attr >> 16), uint8(attr >> 24)}...)
	} else if attr > 255 {
		path = append(path, []uint8{0x31, 0, uint8(attr), uint8(attr >> 8)}...)
	} else if attr >= 0 {
		path = append(path, []uint8{0x30, uint8(attr)}...)
	}
	if member > 65535 {
		path = append(path, []uint8{0x2A, 0, uint8(member), uint8(member >> 8), uint8(member >> 16), uint8(member >> 24)}...)
	} else if member > 255 {
		path = append(path, []uint8{0x29, 0, uint8(member), uint8(member >> 8)}...)
	} else if member >= 0 {
		path = append(path, []uint8{0x28, uint8(member)}...)
	}
	return path
}
