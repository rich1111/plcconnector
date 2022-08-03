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
	attr   []*Tag
	getall []int
	data   []uint8
	m      sync.RWMutex
}

// SetAttr .
func (in *Instance) SetAttr(no int, a *Tag) {
	in.m.Lock()
	in.attr[no] = a
	in.m.Unlock()
}

// SetAttrINT .
func (in *Instance) SetAttrINT(no int, v int16) {
	in.m.Lock()
	binary.LittleEndian.PutUint16(in.attr[no].data, uint16(v))
	in.m.Unlock()
}

// SetAttrDINT .
func (in *Instance) SetAttrDINT(no int, v int32) {
	in.m.Lock()
	binary.LittleEndian.PutUint32(in.attr[no].data, uint32(v))
	in.m.Unlock()
}

// SetAttrUSINT .
func (in *Instance) SetAttrUSINT(no int, v uint8) {
	in.m.Lock()
	in.attr[no].data[0]= v
	in.m.Unlock()
}

// SetAttrUINT .
func (in *Instance) SetAttrUINT(no int, v uint16) {
	in.m.Lock()
	binary.LittleEndian.PutUint16(in.attr[no].data, v)
	in.m.Unlock()
}

// SetAttrUDINT .
func (in *Instance) SetAttrUDINT(no int, v uint32) {
	in.m.Lock()
	binary.LittleEndian.PutUint32(in.attr[no].data, v)
	in.m.Unlock()
}

func (in *Instance) getAttrData(no int) []byte {
	in.m.RLock()
	defer in.m.RUnlock()
	return in.attr[no].DataBytes()
}

func (in *Instance) getAttrAll() []byte {
	var buf bytes.Buffer
	in.m.RLock()
	if in.getall != nil {
		for _, a := range in.getall {
			if in.attr[a] != nil {
				buf.Write(in.attr[a].DataBytes())
			}
		}
	} else {
		for a := 1; a < len(in.attr); a++ {
			if in.attr[a] != nil {
				buf.Write(in.attr[a].DataBytes())
			}
		}
	}
	in.m.RUnlock()
	return buf.Bytes()
}

func (in *Instance) getAttrList(list []int) []byte {
	var buf bytes.Buffer
	in.m.RLock()
	for _, a := range list {
		if in.attr[a] != nil {
			buf.Write(in.attr[a].DataBytes())
		}
	}
	in.m.RUnlock()
	return buf.Bytes()
}

// NewInstance .
func NewInstance(noattr int) *Instance {
	var i Instance
	i.attr = make([]*Tag, noattr+1)
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
	in.attr[1] = TagUINT(1, "Revision")
	in.attr[2] = TagUINT(0, "MaxInstance")
	in.attr[3] = TagUINT(0, "NumInstances")
	in.attr[4] = &Tag{Name: "OptAttrList", data: []uint8{0x00, 0x00}}
	in.attr[5] = &Tag{Name: "OptServiceList", data: []uint8{0x00, 0x00}}
	in.attr[6] = TagUINT(uint16(attrs), "MaxClassAttr")
	in.attr[7] = TagUINT(0, "MaxInstAttr")
	c.inst[0] = in
	return &c
}

// GetClassInstancesList .
func (p *PLC) GetClassInstancesList(class int, instanceFrom int, maxInstances int) ([]int, []*Instance) {
	c, cok := p.Class[class]
	if cok {
		if instanceFrom <= 0 {
			instanceFrom = 1
		}
		c.m.RLock()
		ret := make([]int, 0, len(c.inst))
		i := 0
		for in := range c.inst {
			if in >= instanceFrom {
				ret = append(ret, in)
				i++
				if maxInstances != 0 && i == maxInstances {
					break
				}
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

// GetClassInstanceAttr .
func (p *PLC) GetClassInstanceAttr(class int, instance int, attr int) (*Tag, bool, bool) {
	var (
		aok bool
		at  *Tag
	)

	in := p.GetClassInstance(class, instance)
	if in != nil {
		in.m.RLock()
		if attr < len(in.attr) && attr >= 0 {
			at = in.attr[attr]
			if at != nil {
				aok = true
			}
		}
		in.m.RUnlock()
	}
	return at, aok, in != nil
}

// SetInstance .
func (c *Class) SetInstance(no int, in *Instance) {
	c.m.Lock()
	c.inst[no] = in
	if no > c.lastInst {
		c.lastInst = no
	}
	c.inst[0].SetAttrUINT(2, uint16(c.lastInst))     // MaxInstance
	c.inst[0].SetAttrUINT(3, uint16(len(c.inst)-1))  // NumInstances
	c.inst[0].SetAttrUINT(7, uint16(len(in.attr)-1)) // MaxInstAttr
	c.m.Unlock()
}

func defaultIdentityClass() *Class {
	c := NewClass("Identity", 0)
	c.inst[0].getall = []int{1, 2, 6, 7}
	i := NewInstance(17)
	i.attr[1] = TagUINT(1, "VendorID")
	i.attr[2] = TagUINT(0x0C, "DeviceType") // communications adapter
	i.attr[3] = TagUINT(65001, "ProductCode")
	i.attr[4] = TagUINT(21, "Revision")
	i.attr[5] = TagUINT(0, "Status")
	i.attr[6] = TagUDINT(1234, "SerialNumber")
	i.attr[7] = TagShortString("MongolPLC", "ProductName")
	i.attr[8] = TagUSINT(3, "State")               // operational
	i.attr[9] = TagUINT(0, "ConfConsistencyValue") // or USINT?
	i.attr[10] = TagUSINT(0, "HeartbeatInterval")
	i.attr[10].write = true
	i.attr[11] = &Tag{Name: "ActiveLanguage", data: []byte{'e', 'n', 'g'}}
	i.attr[11].write = true
	i.attr[12] = &Tag{Name: "SuppLangList", data: []byte{'e', 'n', 'g', 'p', 'o', 'l'}, Dim: [3]int{2, 0, 0}, st: &structData{l: 3}}
	i.attr[13] = TagStringI("", "InternationalProductName")
	i.attr[14] = &Tag{Name: "Semaphore", data: []byte{0, 0, 0, 0, 0, 0, 0, 0}} // UINT VendorNumber, UDINT ClientSerialNumber, ITIME MillisecondTimer
	i.attr[14].write = true
	i.attr[15] = TagStringI("", "AssignedName")
	i.attr[16] = TagStringI("", "AssignedDescription")
	i.attr[17] = TagStringI("", "GeographicLocation")

	c.Name = "Identity"
	c.SetInstance(1, i)

	return c
}

func (p *PLC) CreateDefaultAssemblyClass(inputInstance int, outputInstance int) *Class {
	c := NewClass("Assembly", 0)
	c.inst[0].getall = []int{1, 2, 3, 4, 5, 6, 7}
	c.Name = "Assembly"

	in1 := NewInstance(4)
	in1.attr[4] = TagUINT(0, "Size")
	c.SetInstance(inputInstance, in1)

	in2 := NewInstance(4)
	in2.attr[4] = TagUINT(0, "Size")
	c.SetInstance(outputInstance, in2)

	p.Class[AssemblyClass] = c

	return c
}

func (p *PLC) SetSizeTagForAssemblyClass(instance int, val uint16) {
	p.GetClassInstance(AssemblyClass, instance).SetAttrUINT(4, val)
}