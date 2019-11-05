package plcconnector

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// Class .
type Class struct {
	Name string
	Inst map[int]*Instance

	lastInst int
}

// Instance .
type Instance struct {
	Attr []*Attribute

	data     []uint8
	argUint8 [10]uint8
}

// Attribute .
type Attribute struct {
	Name string
	Type int

	data []uint8
}

// AttrUSINT .
func AttrUSINT(v uint8, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeUSINT
	a.data = []byte{v}
	return &a
}

// AttrUINT .
func AttrUINT(v uint16, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeUINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, v)
	return &a
}

// AttrUDINT .
func AttrUDINT(v uint32, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeUDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, v)
	return &a
}

// AttrINT .
func AttrINT(v int16, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, uint16(v))
	return &a
}

// AttrString .
func AttrString(v string, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeString
	a.data = []byte{byte(len(v)), byte(len(v) >> 8)}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// AttrShortString .
func AttrShortString(v string, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeShortString
	a.data = []byte{byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// AttrStringI . TODO len>255
func AttrStringI(v string, n string) *Attribute {
	var a Attribute
	a.Name = n
	a.Type = TypeStringI
	a.data = []byte{1, 'e', 'n', 'g', 0xDA, 4, 0, byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
}

func (in Instance) getAttrAll() []byte {
	var buf bytes.Buffer
	for a := 1; a < len(in.Attr); a++ {
		if in.Attr[a] != nil {
			buf.Write(in.Attr[a].data)
		}
	}
	return buf.Bytes()
}

// NewInstance .
func NewInstance(noattr int) *Instance {
	var i Instance
	i.Attr = make([]*Attribute, noattr+1)
	return &i
}

// NewClass .
func NewClass(n string, attrs int) *Class {
	var c Class
	c.Name = n
	c.Inst = make(map[int]*Instance)
	c.SetInstance(0, NewInstance(attrs))
	return &c
}

// GetClassInstancesList . TODO instanceFrom
func (p *PLC) GetClassInstancesList(class int, instanceFrom int) (*Class, []int) {
	c, cok := p.Class[class]
	if cok {
		ret := make([]int, len(c.Inst)-1) // FIXME instance 0
		i := 0
		for in := range c.Inst {
			if in != 0 { // FIXME instance 0
				ret[i] = in
				i++
			}
		}
		sort.Ints(ret)
		return c, ret
	}
	return nil, nil
}

// GetClassInstance .
func (p *PLC) GetClassInstance(class int, instance int) (*Class, *Instance) {
	c, cok := p.Class[class]
	if cok {
		in, iok := c.Inst[instance]
		if iok {
			return c, in
		}
	}
	return nil, nil
}

// SetInstance .
func (c *Class) SetInstance(no int, in *Instance) {
	c.Inst[no] = in
	if no > c.lastInst {
		c.lastInst = no
	}
}

func defaultIdentityClass() *Class {
	c := NewClass("Identity", 0)
	i := NewInstance(7)
	i.Attr[1] = AttrUINT(1, "VendorID")
	i.Attr[2] = AttrUINT(0x0C, "DeviceType") // communications adapter
	i.Attr[3] = AttrUINT(65001, "ProductCode")
	i.Attr[4] = AttrUINT(1+2<<8, "Revision")
	i.Attr[5] = AttrUINT(0, "Status")
	i.Attr[6] = AttrUDINT(1234, "SerialNumber")
	i.Attr[7] = AttrShortString("MongolPLC", "ProductName")

	c.Name = "Identity"
	c.SetInstance(1, i)

	return c
}
