package plcconnector

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
)

// T .
type T struct {
	N string
	T string
	C int
	O int
}

// NewUDT .
func (p *PLC) NewUDT(udt []T, name string) error {
	p.newUDT(udt, name, 0, 0)
	return nil
}

func (p *PLC) newUDT(udt []T, name string, handle int, size int) error {
	var typencstr bytes.Buffer
	st := new(structData)
	st.o = make(map[string]int)
	st.n = name
	typencstr.WriteString(name)
	typencstr.WriteRune(',')

	st.d = make([]Tag, len(udt))
	for i := 0; i < len(udt); i++ {
		st.d[i].Name = udt[i].N
		st.o[udt[i].N] = i
		if udt[i].C == 0 {
			udt[i].C = 1
		}
		st.d[i].Count = udt[i].C
		st.d[i].Type = p.stringToType(udt[i].T)
		if st.d[i].Type >= TypeStructHead {
			ste, ok := p.tids[udt[i].T]
			if ok {
				st.d[i].st = &ste
			} else {
				panic("!" + udt[i].T)
			}
		} else if st.d[i].Count > 1 {
			st.d[i].Type |= TypeArray1D
		}
		// fmt.Println(udt[i].T, st.d[i].Type)
		typencstr.WriteString(udt[i].T)
		if st.d[i].Type&TypeArray3D > 0 {
			typencstr.WriteString("[" + strconv.Itoa(st.d[i].Count) + "]")
		}
		if i < len(udt)-1 {
			typencstr.WriteRune(',')
		}
		if udt[i].O == 0 {
			st.d[i].offset = st.l
			st.l += st.d[i].ElemLen() * st.d[i].Count
		} else {
			st.d[i].offset = udt[i].O
		}
	}
	if handle == 0 {
		st.h = crc16(typencstr.Bytes())
	} else {
		st.h = uint16(handle)
		st.l = size
	}
	p.addUDT(st)
	fmt.Printf("%v = 0x%X (%d)\n", typencstr.String(), st.h, st.h)
	return nil
}

func (p *PLC) addUDT(st *structData) int {
	p.tMut.Lock()
	ste, ok := p.tids[st.n]
	if ok {
		p.tMut.Unlock()
		return ste.i
	}
	st.i = p.tidLast
	p.tids[st.n] = *st
	p.tidLast++
	p.tMut.Unlock()

	var tp *Instance
	var buf bytes.Buffer

	for _, x := range st.d {
		if x.Count > 1 { // TODO BOOL
			bwrite(&buf, uint16(x.Count))
		} else {
			bwrite(&buf, uint16(0))
		}
		if x.Type >= TypeStructHead {
			if x.st.i > 1 {
				bwrite(&buf, uint16(x.st.i|TypeStruct|TypeArray1D))
			} else {
				bwrite(&buf, uint16(x.st.i|TypeStruct))
			}
		} else {
			bwrite(&buf, uint16(x.Type)) // member type
		}
		bwrite(&buf, uint32(x.offset))
	}
	bwrite(&buf, []byte(st.n+";n\x00")) // template name
	for _, x := range st.d {
		bwrite(&buf, []byte(x.Name+"\x00")) // member name
	}

	bwrite(&buf, make([]byte, (4-buf.Len())&3))

	// fmt.Println(t.st.n, t.st.l, buf.Len())

	tp = NewInstance(5)
	tp.data = buf.Bytes()
	tp.attr[1] = TagUINT(st.h, "StructureHandle")
	tp.attr[2] = TagUINT(uint16(len(st.d)), "TemplateMemberCount")
	tp.attr[3] = TagUINT(uint16(st.l), "UnkownAttr3")                               // the same number as attr5?
	tp.attr[4] = TagUDINT((uint32(buf.Len())+20)/4, "TemplateObjectDefinitionSize") // (x * 4) - 20 // 23 in pdf, was 16
	tp.attr[5] = TagUDINT(uint32(st.l), "TemplateStructureSize")

	p.tMut.Lock()
	p.template.SetInstance(st.i, tp)
	p.tMut.Unlock()
	return st.i
}

func (p *PLC) structHelper(a *Tag, t reflect.Type, fs int, ln int) {
	var typencstr bytes.Buffer
	a.Count = ln
	a.st = new(structData)
	a.st.o = make(map[string]int)
	a.st.n = t.Name()
	typencstr.WriteString(a.st.n)
	typencstr.WriteRune(',')
	a.st.d = make([]Tag, fs)
	for i := 0; i < fs; i++ {
		a.st.d[i].Name = t.Field(i).Name
		a.st.o[t.Field(i).Name] = i
		e := t.Field(i).Type
		a.st.d[i].Count = 1
		a.st.d[i].Type = kindToType(e.Kind())
		if a.st.d[i].Type == TypeArray1D {
			a.st.d[i].Count = e.Len()
			a.st.d[i].Type = TypeArray1D | kindToType(e.Elem().Kind())
		}
		typencstr.WriteString(typeToString(a.st.d[i].Type & TypeType))
		if a.st.d[i].Type&TypeArray3D > 0 {
			typencstr.WriteString("[" + strconv.Itoa(a.st.d[i].Count) + "]")
		}
		if i < fs-1 {
			typencstr.WriteRune(',')
		}
		a.st.d[i].offset = a.st.l
		a.st.l += a.st.d[i].ElemLen() * a.st.d[i].Count
	}
	a.st.h = crc16(typencstr.Bytes())
	a.Type = TypeStructHead | int(a.st.h)
	a.st.i = p.addUDT(a.st)
	fmt.Printf("%v = 0x%X (%d)\n", typencstr.String(), a.st.h, a.st.h)
}
