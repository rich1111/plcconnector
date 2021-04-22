package plcconnector

import (
	"reflect"
	"testing"
)

func Test_parsePath(t *testing.T) {
	tests := []struct {
		name string
		args string
		want []pathEl
	}{
		{"01", "[]", nil},
		{"02", ".", nil},
		{"03", "123", nil},
		{"04", "t[]", nil},
		{"04", "t[ ]", nil},
		{"04", "t[ , ]", nil},
		{"04", "t[, ]", nil},
		{"05", "ta[.ss[2]", nil},
		{"06", "tag1]ss[2]", nil},
		{"07", "tag12[3].ss[2][", nil},
		{"08", "tag123[3].ss[2]]", nil},
		{"09", "tag1234[3].ss[2].", nil},
		{"07", "tag12[3].ss[2][2", nil},
		{"08", "tag123[3].ss[2]]3", nil},
		{"09", "tag1234[3].ss[2].4aa", nil},
		{"10", "4tag", nil},
		{"12", "tag3.5count", nil},
		{"12", "tag3.count.", nil},
		{"12", "tag3 .count[", nil},
		{"12", "tag3 . count[", nil},
		{"12", "ta g3.count[", nil},
		{"12", "tag3.count[", nil},
		{"12", "tag3,count", nil},
		{"12", "tag3.count[,]", nil},
		{"12", "tag3.count,", nil},
		{"12", "tag3.count]", nil},
		{"17", "ta g1ss[2]", nil},
		{"18", "ta'g1ss[2]", nil},
		{"12", "tag3[.].count", nil},
		{"12", "tag3[abc].count", nil},
		{"10", "tag.30.a", nil},
		{"10", "tag.30[12]", nil},
		{"10", "tag.30[12].a.1", nil},
		{"10", "tag:8:U", []pathEl{{typ: ansiExtended, txt: "tag:8:U"}}},
		{"10", "tag_aa:12:33", []pathEl{{typ: ansiExtended, txt: "tag_aa:12:33"}}},
		{"10", "tag", []pathEl{{typ: ansiExtended, txt: "tag"}}},
		{"10", "tag.1", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathBit, val: 1}}},
		{"10", "tag.31", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathBit, val: 31}}},
		{"11", "tag[41].2", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathBit, val: 2}}},
		{"12", "tag3.count.10", []pathEl{{typ: ansiExtended, txt: "tag3"}, {typ: ansiExtended, txt: "count"}, {typ: pathBit, val: 10}}},
		{"11", "tag[41]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}}},
		{"11", "tag[  41]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}}},
		{"11", "tag[ 41  ]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}}},
		{"11", "tag[41  ]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}}},
		{"11", "tag[41][1]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 1}}},
		{"11", "tag[41][11][2]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 11}, {typ: pathMember, val: 2}}},
		{"11", "tag[41][11][2].x", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 11}, {typ: pathMember, val: 2}, {typ: ansiExtended, txt: "x"}}},
		{"11", "tag[41][11][2].1", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 11}, {typ: pathMember, val: 2}, {typ: pathBit, val: 1}}},
		{"11", "tag[41,1]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 1}}},
		{"11", "tag[ 41, 1]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 1}}},
		{"11", "tag[41 ,1]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 1}}},
		{"11", "tag[41 , 1]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 1}}},
		{"11", "tag[41,11,2]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 11}, {typ: pathMember, val: 2}}},
		{"11", "tag[41,11,2].x", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 11}, {typ: pathMember, val: 2}, {typ: ansiExtended, txt: "x"}}},
		{"11", "tag[41,11,2].1", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 41}, {typ: pathMember, val: 11}, {typ: pathMember, val: 2}, {typ: pathBit, val: 1}}},
		{"12", "tag3.count", []pathEl{{typ: ansiExtended, txt: "tag3"}, {typ: ansiExtended, txt: "count"}}},
		{"13", "tag3[5].count", []pathEl{{typ: ansiExtended, txt: "tag3"}, {typ: pathMember, val: 5}, {typ: ansiExtended, txt: "count"}}},
		{"14", "tag.count[712]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: ansiExtended, txt: "count"}, {typ: pathMember, val: 712}}},
		{"15", "tag[6].count[7]", []pathEl{{typ: ansiExtended, txt: "tag"}, {typ: pathMember, val: 6}, {typ: ansiExtended, txt: "count"}, {typ: pathMember, val: 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePath(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePath(\"%s\") = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func Test_constructPath(t *testing.T) {
	tests := []struct {
		name string
		args string
		want []uint8
	}{
		{"10", "tag", []uint8{0x91, 3, 't', 'a', 'g', 0}},
		{"11", "tag[41]", []uint8{0x91, 3, 't', 'a', 'g', 0, 0x28, 41}},
		{"10", "tag.1", []uint8{0x91, 3, 't', 'a', 'g', 0}},
		{"11", "tag[41].2", []uint8{0x91, 3, 't', 'a', 'g', 0, 0x28, 41}},
		{"12", "tag3.count", []uint8{0x91, 4, 't', 'a', 'g', '3', 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"13", "tag3[60000].count", []uint8{0x91, 4, 't', 'a', 'g', '3', 0x29, 0, 0x60, 0xEA, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"13", "tag3[70000].count", []uint8{0x91, 4, 't', 'a', 'g', '3', 0x2A, 0, 112, 17, 1, 0, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"14", "tag.count[712]", []uint8{0x91, 3, 't', 'a', 'g', 0, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0, 0x29, 0, 0xC8, 2}},
		{"15", "tag[6].count[7]", []uint8{0x91, 3, 't', 'a', 'g', 0, 0x28, 6, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0, 0x28, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructPath(parsePath(tt.args)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructPath(\"%s\") = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
