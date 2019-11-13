package plcconnector

import (
	"encoding/binary"
	"math"
)

// Tag .
type Tag struct {
	Name  string
	Type  int
	Index int
	Count int

	data []uint8
}

// TagUSINT .
func TagUSINT(v uint8, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeUSINT
	a.data = []byte{v}
	return &a
}

// TagUINT .
func TagUINT(v uint16, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeUINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, v)
	return &a
}

// TagUDINT .
func TagUDINT(v uint32, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeUDINT
	a.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(a.data, v)
	return &a
}

// TagINT .
func TagINT(v int16, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeINT
	a.data = make([]byte, 2)
	binary.LittleEndian.PutUint16(a.data, uint16(v))
	return &a
}

// TagString .
func TagString(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeString
	a.data = []byte{byte(len(v)), byte(len(v) >> 8)}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// TagShortString .
func TagShortString(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeShortString
	a.data = []byte{byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
}

// TagStringI . TODO len>255
func TagStringI(v string, n string) *Tag {
	var a Tag
	a.Name = n
	a.Type = TypeStringI
	a.data = []byte{1, 'e', 'n', 'g', 0xDA, 4, 0, byte(len(v))}
	a.data = append(a.data, []byte(v)...)
	return &a
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
