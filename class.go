package plcconnector

import (
	"bytes"
	"encoding/binary"
)

// Class .
type Class struct {
	Name string
	Inst map[int]Instance
}

// Instance .
type Instance struct {
	Attr []Attribute
}

// Attribute .
type Attribute struct {
	Name string
	Type int

	data []uint8
}

// productName := []byte{77, 111, 110, 103, 111, 108, 80, 76, 67}

// 					resp.Service = protd.Service + 128
// 					resp.VendorID = 1
// 					resp.DeviceType = 0x0C // communications adapter
// 					resp.ProductCode = 65001
// 					resp.Revision[0] = 1
// 					resp.Revision[1] = 0
// 					resp.Status = 0 // Owned
// 					resp.SerialNumber = 1
// 					resp.ProductNameLength = uint8(len(productName))

// AttrUINT .
func AttrUINT(v uint16, n string) Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeUINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, v)
	return a
}

// AttrUDINT .
func AttrUDINT(v uint32, n string) Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeUDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, v)
	return a
}

// AttrShortString .
func AttrShortString(v string, n string) Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeUDINT
	a.data = []byte{byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return a
}

func (in Instance) getAttrAll() ([]byte, int) {
	var buf bytes.Buffer
	for _, a := range in.Attr {
		buf.Write(a.data)
	}
	return buf.Bytes(), buf.Len()
}

func defaultIdentityClass() Class {
	var (
		c Class
		i Instance
	)
	i.Attr = make([]Attribute, 8)
	i.Attr[1] = AttrUINT(1, "VendorID")
	i.Attr[2] = AttrUINT(0x0C, "DeviceType") // communications adapter
	i.Attr[3] = AttrUINT(65001, "ProductCode")
	i.Attr[4] = AttrUINT(1+2<<8, "Revision")
	i.Attr[5] = AttrUINT(0, "Status")
	i.Attr[6] = AttrUDINT(1234, "SerialNumber")
	i.Attr[7] = AttrShortString("MongolPLC", "ProductName")

	c.Name = "Identity"
	c.Inst = make(map[int]Instance)
	c.Inst[1] = i

	return c
}
