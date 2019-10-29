package plcconnector

import (
	"bytes"
	"encoding/binary"
	"strconv"
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
	a.Type = TypeShortString
	a.data = []byte{byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return a
}

// AttrStringI .
func AttrStringI(v string, n string) Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeStringI
	a.data = []byte{1, 'e', 'n', 'g', 0xDA, 4, 0, byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return a
}

func (in Instance) getAttrAll() ([]byte, int) {
	var buf bytes.Buffer
	for a := 1; a < len(in.Attr); a++ {
		buf.Write(in.Attr[a].data)
	}
	return buf.Bytes(), buf.Len()
}

// NewInstance .
func NewInstance(noattr int) Instance {
	var i Instance
	i.Attr = make([]Attribute, noattr+1)
	for a := range i.Attr {
		i.Attr[a] = AttrUINT(0, "attr_"+strconv.Itoa(a))
	}
	return i
}

// NewClass .
func NewClass(n string, attrs int) Class {
	var c Class
	c.Name = n
	c.Inst = make(map[int]Instance)
	c.Inst[0] = NewInstance(attrs)
	return c
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
