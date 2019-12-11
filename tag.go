package plcconnector

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
)

type structData struct {
	d []Tag
	o map[string]int
	n string // name
	l int    // length
	h uint16 // handle
	i int    // instance (Symbols)
}

// Tag .
type Tag struct {
	Name  string
	Type  int
	Index int
	Count int

	data   []uint8
	st     *structData
	offset int
}

func (st structData) Elem(n string) *Tag {
	i, ok := st.o[n]
	if !ok {
		return nil
	}
	return &st.d[i]
}

// Len .
func (t Tag) Len() int {
	if t.Type >= TypeStructHead {
		return t.st.l
	}
	switch t.Type {
	case TypeSTRING, TypeSTRING2, TypeSTRINGI, TypeSTRINGN, TypeSHORTSTRING:
		return len(t.data)
	default:
		return int(typeLen(uint16(t.Type)))
	}
}

// ElemLen .
func (t Tag) ElemLen() int {
	if t.Type >= TypeStructHead {
		return t.st.l
	}
	return int(typeLen(uint16(t.Type)))
}

// TypeString .
func (t Tag) TypeString() string {
	if t.Type >= TypeStructHead {
		return t.st.n
	}
	return typeToString(t.Type)
}

// TagBOOL .
func TagBOOL(v bool, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeBOOL
	if v {
		a.data = []byte{0xFF}
	} else {
		a.data = []byte{0}
	}
	return &a
}

// TagArrayBool .
func TagArrayBool(v []bool, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeBOOL
	a.data = make([]byte, c)
	for i, x := range v {
		if x {
			a.data[i] = 0xFF
		}
	}
	return &a
}

// TagSINT .
func TagSINT(v int8, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSINT
	a.data = []byte{uint8(v)}
	return &a
}

// TagArraySINT .
func TagArraySINT(v []int8, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeSINT
	a.data = make([]byte, c)
	for i := 0; i < c; i++ {
		a.data[i] = uint8(v[i])
	}
	return &a
}

// TagUSINT .
func TagUSINT(v uint8, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeUSINT
	a.data = []byte{v}
	return &a
}

// TagArrayUSINT .
func TagArrayUSINT(v []uint8, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeUSINT
	a.data = make([]byte, c)
	for i := 0; i < c; i++ {
		a.data[i] = v[i]
	}
	return &a
}

// TagINT .
func TagINT(v int16, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, uint16(v))
	return &a
}

// TagArrayINT .
func TagArrayINT(v []int16, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeINT
	a.data = make([]byte, 2*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint16(a.data[2*i:], uint16(v[i]))
	}
	return &a
}

// TagUINT .
func TagUINT(v uint16, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeUINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, v)
	return &a
}

// TagArrayUINT .
func TagArrayUINT(v []uint16, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeUINT
	a.data = make([]byte, 2*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint16(a.data[2*i:], v[i])
	}
	return &a
}

// TagDINT .
func TagDINT(v int32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, uint32(v))
	return &a
}

// TagArrayDINT .
func TagArrayDINT(v []int32, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeDINT
	a.data = make([]byte, 4*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint32(a.data[4*i:], uint32(v[i]))
	}
	return &a
}

// TagUDINT .
func TagUDINT(v uint32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeUDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, v)
	return &a
}

// TagArrayUDINT .
func TagArrayUDINT(v []uint32, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeUDINT
	a.data = make([]byte, 4*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint32(a.data[4*i:], v[i])
	}
	return &a
}

// TagLINT .
func TagLINT(v int64, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeLINT
	a.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(a.data, uint64(v))
	return &a
}

// TagArrayLINT .
func TagArrayLINT(v []int64, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeLINT
	a.data = make([]byte, 8*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint64(a.data[8*i:], uint64(v[i]))
	}
	return &a
}

// TagULINT .
func TagULINT(v uint64, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeULINT
	a.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(a.data, v)
	return &a
}

// TagArrayULINT .
func TagArrayULINT(v []uint64, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeULINT
	a.data = make([]byte, 8*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint64(a.data[8*i:], v[i])
	}
	return &a
}

// TagREAL .
func TagREAL(v float32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeREAL
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, math.Float32bits(v))
	return &a
}

// TagArrayREAL .
func TagArrayREAL(v []float32, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeREAL
	a.data = make([]byte, 4*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint32(a.data[4*i:], math.Float32bits(v[i]))
	}
	return &a
}

// TagLREAL .
func TagLREAL(v float64, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeLREAL
	a.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(a.data, math.Float64bits(v))
	return &a
}

// TagArrayLREAL .
func TagArrayLREAL(v []float64, c int, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = c
	a.Type = TypeLREAL
	a.data = make([]byte, 8*c)
	for i := 0; i < c; i++ {
		binary.LittleEndian.PutUint64(a.data[8*i:], math.Float64bits(v[i]))
	}
	return &a
}

// TagString .
func TagString(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSTRING
	a.data = []byte{byte(len(v)), byte(len(v) >> 8)}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// TagShortString .
func TagShortString(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSHORTSTRING
	a.data = []byte{byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// TagStringI .
func TagStringI(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Count = 1
	a.Type = TypeSTRINGI
	a.data = []byte{1, 'e', 'n', 'g', TypeSHORTSTRING, 4, 0, byte(len(v))}
	if len(v) > 255 {
		a.data[4] = TypeSTRING
		a.data = append(a.data, byte(len(v)>>8))
	}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// NewTag .
func (p *PLC) NewTag(i interface{}, n string) {
	var a *Tag
	r := reflect.TypeOf(i)
	v := reflect.ValueOf(i)
	switch r.Kind() {
	case reflect.Bool:
		a = TagBOOL(v.Bool(), n)
	case reflect.Int8:
		a = TagSINT(int8(v.Int()), n)
	case reflect.Int16:
		a = TagINT(int16(v.Int()), n)
	case reflect.Int32:
		a = TagDINT(int32(v.Int()), n)
	case reflect.Int64:
		a = TagLINT(v.Int(), n)
	case reflect.Uint8:
		a = TagUSINT(uint8(v.Uint()), n)
	case reflect.Uint16:
		a = TagUINT(uint16(v.Uint()), n)
	case reflect.Uint32:
		a = TagUDINT(uint32(v.Int()), n)
	case reflect.Uint64:
		a = TagULINT(v.Uint(), n)
	case reflect.Float32:
		a = TagREAL(float32(v.Float()), n)
	case reflect.Float64:
		a = TagLREAL(v.Float(), n)
	case reflect.Array, reflect.Slice:
		e := r.Elem()
		l := v.Len()
		switch e.Kind() {
		case reflect.Bool:
			bytes := make([]bool, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Bool()
			}
			a = TagArrayBool(bytes, l, n)
		case reflect.Int8:
			bytes := make([]int8, l)
			for i := range bytes {
				bytes[i] = int8(v.Index(i).Int())
			}
			a = TagArraySINT(bytes, l, n)
		case reflect.Int16:
			bytes := make([]int16, l)
			for i := range bytes {
				bytes[i] = int16(v.Index(i).Int())
			}
			a = TagArrayINT(bytes, l, n)
		case reflect.Int32:
			bytes := make([]int32, l)
			for i := range bytes {
				bytes[i] = int32(v.Index(i).Int())
			}
			a = TagArrayDINT(bytes, l, n)
		case reflect.Int64:
			bytes := make([]int64, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Int()
			}
			a = TagArrayLINT(bytes, l, n)
		case reflect.Uint8:
			bytes := make([]uint8, l)
			for i := range bytes {
				bytes[i] = uint8(v.Index(i).Uint())
			}
			a = TagArrayUSINT(bytes, l, n)
		case reflect.Uint16:
			bytes := make([]uint16, l)
			for i := range bytes {
				bytes[i] = uint16(v.Index(i).Uint())
			}
			a = TagArrayUINT(bytes, l, n)
		case reflect.Uint32:
			bytes := make([]uint32, l)
			for i := range bytes {
				bytes[i] = uint32(v.Index(i).Uint())
			}
			a = TagArrayUDINT(bytes, l, n)
		case reflect.Uint64:
			bytes := make([]uint64, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Uint()
			}
			a = TagArrayULINT(bytes, l, n)
		case reflect.Float32:
			bytes := make([]float32, l)
			for i := range bytes {
				bytes[i] = float32(v.Index(i).Float())
			}
			a = TagArrayREAL(bytes, l, n)
		case reflect.Float64:
			bytes := make([]float64, l)
			for i := range bytes {
				bytes[i] = v.Index(i).Float()
			}
			a = TagArrayLREAL(bytes, l, n)
		case reflect.Struct:
			a = new(Tag)
			a.Name = n
			p.structHelper(a, e, e.NumField(), l)
			a.data = make([]uint8, 0, a.st.l*l)
			for i := 0; i < l; i++ {
				for j := 0; j < e.NumField(); j++ {
					a.data = append(a.data, valueToByte(v.Index(i).Field(j))...)
				}
			}
		default:
			panic("unsupported embedded type " + e.String())
		}
	case reflect.String:
		a = TagString(v.String(), n)
	case reflect.Struct:
		a = new(Tag)
		a.Name = n
		p.structHelper(a, r, v.NumField(), 1)
		a.data = make([]uint8, 0, a.st.l)
		for i := 0; i < v.NumField(); i++ {
			a.data = append(a.data, valueToByte(v.Field(i))...)
		}
	default:
		panic("unknown type " + r.String())
	}
	p.AddTag(*a)
}

func valueToByte(v reflect.Value) []byte {
	var r []byte
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			r = []byte{0xFF}
		} else {
			r = []byte{0}
		}
	case reflect.Int8:
		r = []byte{uint8(v.Int())}
	case reflect.Int16:
		r = make([]byte, 2)
		binary.LittleEndian.PutUint16(r, uint16(v.Int()))
	case reflect.Int32:
		r = make([]byte, 4)
		binary.LittleEndian.PutUint32(r, uint32(v.Int()))
	case reflect.Int64:
		r = make([]byte, 8)
		binary.LittleEndian.PutUint64(r, uint64(v.Int()))
	case reflect.Uint8:
		r = []byte{uint8(v.Uint())}
	case reflect.Uint16:
		r = make([]byte, 2)
		binary.LittleEndian.PutUint16(r, uint16(v.Uint()))
	case reflect.Uint32:
		r = make([]byte, 4)
		binary.LittleEndian.PutUint32(r, uint32(v.Uint()))
	case reflect.Uint64:
		r = make([]byte, 8)
		binary.LittleEndian.PutUint64(r, v.Uint())
	case reflect.Float32:
		r = make([]byte, 4)
		binary.LittleEndian.PutUint32(r, math.Float32bits(float32(v.Float())))
	case reflect.Float64:
		r = make([]byte, 8)
		binary.LittleEndian.PutUint64(r, math.Float64bits(v.Float()))
	case reflect.Array:
		e := v.Index(0)
		c := v.Len()
		switch e.Kind() {
		case reflect.Bool:
			r = make([]byte, c)
			for i := 0; i < c; i++ {
				if v.Index(i).Bool() {
					r[i] = 0xFF
				}
			}
		case reflect.Int8:
			r = make([]byte, c)
			for i := 0; i < c; i++ {
				r[i] = uint8(v.Index(i).Int())
			}
		case reflect.Int16:
			r = make([]byte, 2*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint16(r[2*i:], uint16(v.Index(i).Int()))
			}
		case reflect.Int32:
			r = make([]byte, 4*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint32(r[4*i:], uint32(v.Index(i).Int()))
			}
		case reflect.Int64:
			r = make([]byte, 8*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint64(r[8*i:], uint64(v.Index(i).Int()))
			}
		case reflect.Uint8:
			r = make([]byte, c)
			for i := 0; i < c; i++ {
				r[i] = uint8(v.Index(i).Uint())
			}
		case reflect.Uint16:
			r = make([]byte, 2*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint16(r[2*i:], uint16(v.Index(i).Uint()))
			}
		case reflect.Uint32:
			r = make([]byte, 4*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint32(r[4*i:], uint32(v.Index(i).Uint()))
			}
		case reflect.Uint64:
			r = make([]byte, 8*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint64(r[8*i:], v.Index(i).Uint())
			}
		case reflect.Float32:
			r = make([]byte, 4*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint32(r[4*i:], math.Float32bits(float32(v.Index(i).Float())))
			}
		case reflect.Float64:
			r = make([]byte, 8*c)
			for i := 0; i < c; i++ {
				binary.LittleEndian.PutUint64(r[8*i:], math.Float64bits(v.Index(i).Float()))
			}
		default:
			panic("unsupported embedded value type " + e.String())
		}
	default:
		panic("unsupported value type " + v.String())
	}
	return r
}

func kindToType(k reflect.Kind) int {
	switch k {
	case reflect.Bool:
		return TypeBOOL
	case reflect.Int8:
		return TypeSINT
	case reflect.Int16:
		return TypeINT
	case reflect.Int32:
		return TypeDINT
	case reflect.Int64:
		return TypeLINT
	case reflect.Uint8:
		return TypeUSINT
	case reflect.Uint16:
		return TypeUINT
	case reflect.Uint32:
		return TypeUDINT
	case reflect.Uint64:
		return TypeULINT
	case reflect.Float32:
		return TypeREAL
	case reflect.Float64:
		return TypeLREAL
	case reflect.Array:
		return TypeArray1D
	default:
		panic("unsupported kind type " + k.String())
	}
}

// DataBytes returns array of bytes.
func (t *Tag) DataBytes() []byte {
	return t.data
}

// DataBOOL returns array of BOOL.
func (t *Tag) DataBOOL() []bool {
	ret := make([]bool, 0, t.Count)
	for i := 0; i < len(t.data); i++ {
		tmp := false
		if t.data[i] != 0 {
			tmp = true
		}
		ret = append(ret, tmp)
	}
	return ret
}

// DataSINT returns array of int8.
func (t *Tag) DataSINT() []int8 {
	ret := make([]int8, 0, t.Count)
	for i := 0; i < len(t.data); i++ {
		ret = append(ret, int8(t.data[i]))
	}
	return ret
}

// DataINT returns array of int16.
func (t *Tag) DataINT() []int16 {
	ret := make([]int16, 0, t.Count)
	for i := 0; i < len(t.data); i += 2 {
		tmp := int16(t.data[i])
		tmp += int16(t.data[i+1]) << 8
		ret = append(ret, tmp)
	}
	return ret
}

// DataDINT returns array of int32.
func (t *Tag) DataDINT() []int32 {
	ret := make([]int32, 0, t.Count)
	for i := 0; i < len(t.data); i += 4 {
		tmp := int32(t.data[i])
		tmp += int32(t.data[i+1]) << 8
		tmp += int32(t.data[i+2]) << 16
		tmp += int32(t.data[i+3]) << 24
		ret = append(ret, tmp)
	}
	return ret
}

// DataREAL returns array of float32.
func (t *Tag) DataREAL() []float32 {
	ret := make([]float32, 0, t.Count)
	for i := 0; i < len(t.data); i += 4 {
		tmp := uint32(t.data[i])
		tmp += uint32(t.data[i+1]) << 8
		tmp += uint32(t.data[i+2]) << 16
		tmp += uint32(t.data[i+3]) << 24
		ret = append(ret, math.Float32frombits(tmp))
	}
	return ret
}

// DataDWORD returns array of int32.
func (t *Tag) DataDWORD() []int32 {
	return t.DataDINT()
}

// DataLINT returns array of int64.
func (t *Tag) DataLINT() []int64 {
	ret := make([]int64, 0, t.Count)
	for i := 0; i < len(t.data); i += 8 {
		tmp := int64(t.data[i])
		tmp += int64(t.data[i+1]) << 8
		tmp += int64(t.data[i+2]) << 16
		tmp += int64(t.data[i+3]) << 24
		tmp += int64(t.data[i+4]) << 32
		tmp += int64(t.data[i+5]) << 40
		tmp += int64(t.data[i+6]) << 48
		tmp += int64(t.data[i+7]) << 56
		ret = append(ret, tmp)
	}
	return ret
}

// DataString returns string.
func (t *Tag) DataString() string {
	switch t.Type {
	case TypeSTRING:
		return string(t.data[2:])
	case TypeSHORTSTRING:
		return string(t.data[1:])
	default:
		fmt.Println("DataString error", t.Type)
	}
	return "error string"
}

// CreateTag .
func (p *PLC) CreateTag(typ string, name string) {
	var t Tag
	st, ok := p.tids[typ]
	if ok {
		t.st = &st
		t.Type = TypeStructHead | int(t.st.h)
		t.Name = name
		t.Count = 1
		t.data = make([]uint8, t.st.l)
		p.AddTag(t)
	}
}

func (p *PLC) tagError(service int, status int, tag *Tag) {
	if p.callback != nil {
		go p.callback(service, status, tag)
	}
}

func (p *PLC) parsePathEl(path []pathEl) (*Tag, uint32, int, int, int, error) {
	var (
		copyFrom int
		index    int
		memb     string
		pi       = 1
		tag      string
		tgtyp    uint32
		tl       int
	)

	if len(path) == 0 {
		return nil, 0, 0, 0, 0, errors.New("path length 0")
	}

	if path[0].typ == ansiExtended {
		tag = path[0].txt
	} else if len(path) > 1 && path[0].typ == pathClass && path[0].val == SymbolClass && path[1].typ == pathInstance {
		pi = 2
		tag = p.symbols.inst[path[1].val].attr[1].DataString()
	} else {
		return nil, 0, 0, 0, 0, errors.New("path unkown element")
	}

	tg, ok := p.tags[tag]

	if !ok {
		return nil, 0, 0, 0, 0, errors.New("path no tag")
	}

	tl = tg.Len()
	tgtyp = uint32(tg.Type)

	tgc := tg
	for i := pi; i < len(path); i++ {
		switch path[i].typ {
		case pathElement: // TODO: test 2d, 3d array
			index = path[i].val
			copyFrom += index * tl
		case ansiExtended:
			if tgc.st == nil {
				return nil, 0, 0, 0, 0, errors.New("path tag is not a struct")
			}
			memb = path[i].txt
			el := tgc.st.Elem(memb)
			if el == nil {
				fmt.Println("no member", memb, "in struct", tgc.Name)
				return nil, 0, 0, 0, 0, errors.New("path no member in struct")
			}
			tl = el.Len()
			copyFrom += el.offset
			tgtyp = uint32(el.Type) // TODO BOOL
			tgc = el
		}
	}

	if tgc.st == nil {
		tgtyp &= TypeType
	}
	p.debug(tgc.TypeString())

	return tg, tgtyp, tl, copyFrom, index, nil
}

func (p *PLC) readTag(path []pathEl, count uint16) ([]uint8, uint32, int, bool) {
	p.tMut.RLock()
	defer p.tMut.RUnlock()

	tg, tgtyp, tl, copyFrom, index, err := p.parsePathEl(path)

	if err != nil {
		p.tagError(ReadTag, PathSegmentError, nil)
		return nil, 0, 0, false
	}

	copyLen := int(count) * tl
	tgdata := make([]uint8, copyLen)
	if copyFrom+copyLen > len(tg.data) {
		p.tagError(ReadTag, PathSegmentError, nil)
		return nil, 0, 0, false
	}
	copy(tgdata, tg.data[copyFrom:])

	p.tagError(ReadTag, Success, &Tag{Name: tg.Name, Type: int(tgtyp), Index: index, Count: int(count), data: tgdata})
	return tgdata, tgtyp, tl, true
}

func (p *PLC) readModWriteTag(path []pathEl, orMask, andMask []uint8) bool { // FIXME false callback
	var (
		tag   string
		index int
		pi    int
	)

	if len(path) > 0 {
		if path[0].typ == ansiExtended { // TODO better, function
			tag = path[0].txt
		} else if len(path) > 1 && path[0].typ == pathClass && path[0].val == SymbolClass && path[1].typ == pathInstance {
			pi = 1
			tag = p.symbols.inst[path[1].val].attr[1].DataString()
		} else {
			return false
		}
		if len(path) > pi+1 {
			switch path[pi+1].typ {
			case pathElement:
				index = path[pi+1].val
			}
		}
	} else {
		return false
	}

	p.tMut.Lock()
	tg, ok := p.tags[tag]
	if ok && tg.Count >= index {
		el := index * tg.ElemLen()
		for i, or := range orMask {
			tg.data[el+i] |= or
		}
		for i, and := range andMask {
			tg.data[el+i] &= and
		}
	} else {
		p.tMut.Unlock()
		return false
	}
	p.tMut.Unlock()
	// if p.callback != nil { TODO
	// 	go p.callback(WriteTag, Success, &Tag{Name: tag, Type: int(typ), Index: index, Count: count, data: data})
	// }
	return true
}

func (p *PLC) saveTag(path []pathEl, typ uint16, count int, data []uint8, offset int) bool { // FIXME false callback
	var (
		tag   string
		index int
		pi    int
	)

	if len(path) > 0 {
		if path[0].typ == ansiExtended { // TODO better, function
			tag = path[0].txt
		} else if len(path) > 1 && path[0].typ == pathClass && path[0].val == SymbolClass && path[1].typ == pathInstance {
			pi = 1
			tag = p.symbols.inst[path[1].val].attr[1].DataString()
		} else {
			return false
		}
		if len(path) > pi+1 {
			switch path[pi+1].typ {
			case pathElement:
				index = path[pi+1].val
			}
		}
	} else {
		return false
	}

	p.tMut.Lock()
	tg, ok := p.tags[tag]
	index += offset / tg.ElemLen()
	if ok && tg.Type == int(typ) && tg.Count >= index+count {
		copy(tg.data[index*tg.ElemLen():], data)
	} else {
		p.tMut.Unlock()
		return false
	}
	p.tMut.Unlock()
	if p.callback != nil {
		go p.callback(WriteTag, Success, &Tag{Name: tag, Type: int(typ), Index: index, Count: count, data: data})
	}
	return true
}

func (p *PLC) addTag(t Tag, instance int) {
	if t.data == nil {
		size := uint16(t.ElemLen()) * uint16(t.Count)
		t.data = make([]uint8, size)
	}
	in := NewInstance(8)
	in.attr[1] = TagString(t.Name, "SymbolName")
	typ := uint16(t.Type)
	if t.Count > 1 {
		typ |= TypeArray1D
	}
	if t.Type >= TypeStructHead {
		in.attr[2] = TagUINT(TypeStruct+uint16(t.st.i), "SymbolType")
	} else {
		in.attr[2] = TagUINT(typ, "SymbolType")
	}
	in.attr[7] = TagUINT(uint16(t.ElemLen()), "BaseTypeSize")
	if t.Count > 1 {
		in.attr[8] = &Tag{Name: "Dimensions", data: []uint8{uint8(t.Count), uint8(t.Count >> 8), uint8(t.Count >> 16), uint8(t.Count >> 24), 0, 0, 0, 0, 0, 0, 0, 0}}
	} else {
		in.attr[8] = &Tag{Name: "Dimensions", data: []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	}
	p.tMut.Lock()
	if instance == -1 {
		p.symbols.SetInstance(p.symbols.lastInst+1, in)
	} else {
		p.symbols.SetInstance(instance, in)
	}
	p.tags[t.Name] = &t
	p.tMut.Unlock()
}

// AddTag adds tag.
func (p *PLC) AddTag(t Tag) {
	p.addTag(t, -1)
}

// UpdateTag sets data to the tag
func (p *PLC) UpdateTag(name string, offset int, data []uint8) bool {
	p.tMut.Lock()
	defer p.tMut.Unlock()
	t, ok := p.tags[name]
	if !ok {
		fmt.Println("plcconnector UpdateTag: no tag named ", name)
		return false
	}
	offset *= t.ElemLen()
	to := offset + len(data)
	if to > len(t.data) {
		fmt.Println("plcconnector UpdateTag: to large data ", name)
		return false
	}
	for i := offset; i < to; i++ {
		t.data[i] = data[i-offset]
	}
	return true
}
