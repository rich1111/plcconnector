package plcconnector

import (
	"bytes"
	"encoding/binary"
	"sort"
	"sync"
)

// Class .
type Class struct {
	Name string

	inst     map[int]*Instance
	lastInst int
	m        sync.RWMutex
}

// Instance .
type Instance struct {
	attr []*Attribute
	data []uint8
	m    sync.RWMutex
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

// SetAttr .
func (in *Instance) SetAttr(no int, a *Attribute) {
	in.m.Lock()
	in.attr[no] = a
	in.m.Unlock()
}

func (in *Instance) setAttrData(no int, data []byte) {
	in.m.Lock()
	in.attr[no].data = data
	in.m.Unlock()
}

// SetAttrUINT .
func (in *Instance) SetAttrUINT(no int, v uint16) {
	in.m.Lock()
	binary.LittleEndian.PutUint16(in.attr[no].data, v)
	in.m.Unlock()
}

func (in *Instance) getAttrData(no int) []byte {
	in.m.RLock()
	defer in.m.RUnlock()
	return in.attr[no].data
}

func (in *Instance) getAttrAll() []byte {
	var buf bytes.Buffer
	in.m.RLock()
	for a := 1; a < len(in.attr); a++ {
		if in.attr[a] != nil {
			buf.Write(in.attr[a].data)
		}
	}
	in.m.RUnlock()
	return buf.Bytes()
}

// NewInstance .
func NewInstance(noattr int) *Instance {
	var i Instance
	i.attr = make([]*Attribute, noattr+1)
	return &i
}

// NewClass .
func NewClass(n string, attrs int) *Class {
	var c Class
	c.Name = n
	c.inst = make(map[int]*Instance)
	if attrs < 7 {
		attrs = 7
	}
	in := NewInstance(attrs)
	in.attr[1] = AttrUINT(1, "Revision")
	in.attr[2] = AttrUINT(0, "MaxInstance")
	in.attr[3] = AttrUINT(0, "NumInstances")
	c.inst[0] = in
	return &c
}

// GetClassInstancesList .
func (p *PLC) GetClassInstancesList(class int, instanceFrom int) ([]int, []*Instance) {
	c, cok := p.Class[class]
	if cok {
		if instanceFrom <= 0 {
			instanceFrom = 1
		}
		c.m.RLock()
		ret := make([]int, 0, len(c.inst)-instanceFrom)
		i := 0
		for in := range c.inst {
			if in >= instanceFrom {
				ret = append(ret, in)
				i++
			}
		}
		sort.Ints(ret)
		ret2 := make([]*Instance, len(ret))
		for a, b := range ret {
			ret2[a] = c.inst[b]
		}
		c.m.RUnlock()
		return ret, ret2
	}
	return nil, nil
}

// GetClassInstance .
func (p *PLC) GetClassInstance(class int, instance int) *Instance {
	c, cok := p.Class[class]
	if cok {
		c.m.RLock()
		defer c.m.RUnlock()
		in, iok := c.inst[instance]
		if iok {
			p.debug(c.Name, instance)
			return in
		}
	}
	return nil
}

// SetInstance .
func (c *Class) SetInstance(no int, in *Instance) {
	c.m.Lock()
	c.inst[no] = in
	if no > c.lastInst {
		c.lastInst = no
	}
	c.inst[0].SetAttrUINT(2, uint16(c.lastInst))
	c.inst[0].SetAttrUINT(3, uint16(len(c.inst)))
	c.m.Unlock()
}

func defaultIdentityClass() *Class {
	c := NewClass("Identity", 0)
	i := NewInstance(7)
	i.attr[1] = AttrUINT(1, "VendorID")
	i.attr[2] = AttrUINT(0x0C, "DeviceType") // communications adapter
	i.attr[3] = AttrUINT(65001, "ProductCode")
	i.attr[4] = AttrUINT(1+2<<8, "Revision")
	i.attr[5] = AttrUINT(0, "Status")
	i.attr[6] = AttrUDINT(1234, "SerialNumber")
	i.attr[7] = AttrShortString("MongolPLC", "ProductName")

	c.Name = "Identity"
	c.SetInstance(1, i)

	return c
}
