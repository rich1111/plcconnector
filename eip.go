package plcconnector

import (
	"encoding/binary"
	"math/rand"
)

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
