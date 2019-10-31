package plcconnector

import (
	"encoding/binary"
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

func (r *req) parsePath(path []uint8) {
	for i := 0; i < len(path); i++ {
		if path[i] == ansiExtended {
			ln := path[i+1]
			ansi := string(path[i+2 : i+int(ln)+2])
			i += int(ln) + 1
			if ln&1 == 1 {
				i++
			}
			r.p.debug("ansi", ansi)
		} else if (path[i] & pathType) == pathLogical {
			typ := path[i] & pathSegType
			size := path[i] & pathSize
			name := ""
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
				return
			}
			switch typ {
			case pathClass:
				name = "class"
			case pathInstance:
				name = "instance"
			case pathAttribute:
				name = "attribute"
			case pathElement:
				name = "element"
			default:
				r.p.debug("path segment type error")
				return
			}
			r.p.debug(name, el)
		} else {
			r.p.debug("path type error", path[i])
		}
	}
	r.p.debug()
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
	r.parsePath([]uint8{0x91, 0x06, 0x64, 0x77, 0x65, 0x6C, 0x6C, 0x33, 0x91, 0x03, 0x61, 0x63, 0x63, 0x00}) // FIXME
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

	r.encHead.SessionHandle = rand.Uint32()

	r.write(data)
	return nil
}

func (r *req) eipListIdentity() error {
	r.p.debug("ListIdentity")

	itemCount := uint16(1)
	state := uint8(0)
	productName := []byte{77, 111, 110, 103, 111, 108, 80, 76, 67}
	var (
		data listIdentityData
		typ  itemType
	)

	data.ProtocolVersion = 1
	data.SocketFamily = htons(2)
	data.SocketPort = htons(r.p.port)
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

	r.write(itemCount)
	r.write(typ)
	r.write(data)
	r.write(productName)
	r.write(state)
	return nil
}

func (r *req) eipListServices() error {
	r.p.debug("ListServices")

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

	r.write(itemCount)
	r.write(typ)
	r.write(data)
	return nil
}
