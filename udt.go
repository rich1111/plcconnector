package plcconnector

import (
	"bytes"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type udtT struct {
	N  string // Name
	T  string // Type
	C  int    // Count
	C2 int    // Count 2D
	C3 int    // Count 3D
	O  int    // Offset
}

// NewUDT .
func (p *PLC) NewUDT(udt string) error {
	u, n := udtFromString(udt)
	p.newUDT(u, n, 0, 0)
	return nil
}

func (p *PLC) newUDT(udt []udtT, name string, handle int, size int) error {
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
		st.d[i].Dim[0] = udt[i].C
		st.d[i].Dim[1] = udt[i].C2
		st.d[i].Dim[2] = udt[i].C3
		st.d[i].Type = p.stringToType(udt[i].T)
		if st.d[i].Type == 0 {
			panic("!" + udt[i].T)
		}
		if st.d[i].Type >= TypeStructHead {
			ste, ok := p.tids[udt[i].T]
			if ok {
				st.d[i].st = &ste
			} else {
				panic("!" + udt[i].T)
			}
		} else if st.d[i].Dim[2] > 0 {
			st.d[i].Type |= TypeArray3D
		} else if st.d[i].Dim[1] > 0 {
			st.d[i].Type |= TypeArray2D
		} else if st.d[i].Dim[0] > 0 && udt[i].T != "BOOL" {
			st.d[i].Type |= TypeArray1D
		}
		// fmt.Println(udt[i].T, st.d[i].Type)
		typencstr.WriteString(udt[i].T)
		if st.d[i].Type&TypeArray3D > 0 {
			typencstr.WriteString(st.d[i].DimString())
		}
		if i < len(udt)-1 {
			typencstr.WriteRune(',')
		}
		if udt[i].O == -1 {
			st.d[i].offset = st.l
			st.l += st.d[i].ElemLen() * st.d[i].Dims()
		} else {
			st.d[i].offset = udt[i].O
			st.l = udt[i].O + st.d[i].ElemLen()
		}
	}
	if handle == 0 {
		st.h = crc16(typencstr.Bytes())
	} else {
		st.h = uint16(handle)
		st.l = size
	}
	p.addUDT(st)

	in := p.Class[0xAC].inst[1]
	crc := in.attr[4].DataDINT()[0] + int32(crc16([]byte(name))+st.h)
	in.SetAttrDINT(4, crc)

	// fmt.Printf("%v = 0x%X (%d)\n", typencstr.String(), st.h, st.h)
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
		bwrite(&buf, uint16(x.Dim[0]))
		if x.Type >= TypeStructHead {
			bwrite(&buf, uint16(x.st.i|TypeStruct))
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

	// fmt.Println(st.n, st.l, buf.Len())

	tp = NewInstance(5)
	tp.data = buf.Bytes()
	tp.attr[1] = TagUINT(st.h, "StructureHandle")
	tp.attr[2] = TagUINT(uint16(len(st.d)), "TemplateMemberCount")
	tp.attr[3] = TagUINT(uint16(st.l), "UnkownAttr3")                               // the same number as attr5?
	tp.attr[4] = TagUDINT((uint32(buf.Len())+20)/4, "TemplateObjectDefinitionSize") // (x * 4) - 20 // 23 in pdf, was 16
	tp.attr[5] = TagUDINT(uint32(st.l), "TemplateStructureSize")

	p.tMut.Lock()
	p.template.SetInstance(st.i, tp)
	p.template.SetInstance(int(st.h), tp)
	p.tMut.Unlock()
	return st.i
}

func (p *PLC) structHelper(a *Tag, t reflect.Type, fs int, ln int) {
	var typencstr bytes.Buffer
	a.Dim[0] = ln
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
		a.st.d[i].Type = kindToType(e.Kind())
		if a.st.d[i].Type == TypeArray1D {
			a.st.d[i].Dim[0] = e.Len()
			a.st.d[i].Type = TypeArray1D | kindToType(e.Elem().Kind())
		}
		typencstr.WriteString(typeToString(a.st.d[i].Type & TypeType))
		if a.st.d[i].Type&TypeArray3D > 0 {
			typencstr.WriteString(a.st.d[i].DimString())
		}
		if i < fs-1 {
			typencstr.WriteRune(',')
		}
		a.st.d[i].offset = a.st.l
		a.st.l += a.st.d[i].ElemLen() * a.st.d[i].Dims()
	}
	a.st.h = crc16(typencstr.Bytes())
	a.Type = TypeStructHead | int(a.st.h)
	a.st.i = p.addUDT(a.st)
	// fmt.Printf("%v = 0x%X (%d)\n", typencstr.String(), a.st.h, a.st.h)
}

// INT name[1, 2, 3]
//   1    2 3  4  5
var udtr = regexp.MustCompile(`(\w+)\s*(\w*)\s*\[*\s*(\d*)\s*,*\s*(\d*)\s*,*\s*(\d*)\s*\]*`)

func udtFromString(udt string) ([]udtT, string) {
	t := []udtT{}

	if !strings.HasPrefix(udt, "DATATYPE") {
		sb := udtr.FindStringSubmatch(udt)
		tn := udtT{}
		tn.T = sb[1]
		i, err := strconv.Atoi(sb[3])
		if err == nil {
			tn.C = i
		}
		i, err = strconv.Atoi(sb[4])
		if err == nil {
			tn.C2 = i
		}
		i, err = strconv.Atoi(sb[5])
		if err == nil {
			tn.C3 = i
		}

		t = append(t, tn)
		return t, ""
	}
	u := strings.FieldsFunc(udt, func(r rune) bool {
		if unicode.IsSpace(r) || r == ';' || r == '(' || r == ')' {
			return true
		}
		return false
	})
	name := u[1]
	u = u[2:]
	booli := 0
	for i := 0; i < len(u); i += 2 {
		if u[i] == "END_DATATYPE" {
			break
		}
		if u[i] == "FamilyType" || u[i] == "Radix" {
			i++
			continue
		}
		if strings.HasPrefix(u[i], "FamilyType") || strings.HasPrefix(u[i], "Radix") || strings.HasPrefix(u[i], ":=") {
			i--
			continue
		}

		str := u[i] + " " + u[i+1]

		sb := udtr.FindStringSubmatch(str)
		tn := udtT{}
		tn.N = sb[2]
		tn.T = sb[1]
		if tn.T == "BOOL" {
			tn.C = booli
			booli++ // FIXME > 8
		} else {
			tn.O = -1
		}
		i, err := strconv.Atoi(sb[3])
		if err == nil {
			tn.C = i
		}
		i, err = strconv.Atoi(sb[4])
		if err == nil {
			tn.C2 = i
		}
		i, err = strconv.Atoi(sb[5])
		if err == nil {
			tn.C3 = i
		}

		t = append(t, tn)
	}

	return t, name
}
