package plcconnector

import (
	"reflect"
	"testing"
)

func Test_req_parsePath(t *testing.T) {
	type args struct {
		path []uint8
	}
	tests := []struct {
		name    string
		r       *req
		args    args
		want    int
		want1   int
		want2   int
		want3   []pathEl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3, err := tt.r.parsePath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("req.parsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("req.parsePath() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("req.parsePath() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("req.parsePath() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("req.parsePath() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func Test_parsePath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want []pathEl
	}{
		{"01", args{"[]"}, nil},
		{"02", args{"."}, nil},
		{"03", args{"123"}, nil},
		{"04", args{"t[]"}, nil},
		{"05", args{"ta[.ss[2]"}, nil},
		{"06", args{"tag1]ss[2]"}, nil},
		{"07", args{"tag12[3].ss[2]["}, nil},
		{"08", args{"tag123[3].ss[2]]"}, nil},
		{"09", args{"tag1234[3].ss[2]."}, nil},
		{"07", args{"tag12[3].ss[2][2"}, nil},
		{"08", args{"tag123[3].ss[2]]3"}, nil},
		{"09", args{"tag1234[3].ss[2].4aa"}, nil},
		{"10", args{"4tag"}, nil},
		{"12", args{"tag3.5count"}, nil},
		{"12", args{"tag3.count."}, nil},
		{"12", args{"tag3 .count["}, nil},
		{"12", args{"tag3 . count["}, nil},
		{"12", args{"ta g3.count["}, nil},
		{"12", args{"tag3.count["}, nil},
		{"12", args{"tag3,count"}, nil},
		{"12", args{"tag3.count[,]"}, nil},
		{"12", args{"tag3.count,"}, nil},
		{"12", args{"tag3.count]"}, nil},
		{"17", args{"ta g1ss[2]"}, nil},
		{"18", args{"ta'g1ss[2]"}, nil},
		{"12", args{"tag3[.].count"}, nil},
		{"12", args{"tag3[abc].count"}, nil},
		{"10", args{"tag.30.a"}, nil},
		{"10", args{"tag.30[12]"}, nil},
		{"10", args{"tag.30[12].a.1"}, nil},
		{"10", args{"tag:8:U"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag:8:U"}}},
		{"10", args{"tag_aa:12:33"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag_aa:12:33"}}},
		{"10", args{"tag"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}}},
		{"10", args{"tag.1"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathBit, val: 1}}},
		{"10", args{"tag.31"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathBit, val: 31}}},
		{"11", args{"tag[41].2"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathBit, val: 2}}},
		{"12", args{"tag3.count.10"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag3"}, pathEl{typ: ansiExtended, txt: "count"}, pathEl{typ: pathBit, val: 10}}},
		{"11", args{"tag[41]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}}},
		{"11", args{"tag[  41]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}}},
		{"11", args{"tag[ 41  ]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}}},
		{"11", args{"tag[41  ]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}}},
		{"11", args{"tag[41][1]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 1}}},
		{"11", args{"tag[41][11][2]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 11}, pathEl{typ: pathElement, val: 2}}},
		{"11", args{"tag[41][11][2].x"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 11}, pathEl{typ: pathElement, val: 2}, pathEl{typ: ansiExtended, txt: "x"}}},
		{"11", args{"tag[41][11][2].1"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 11}, pathEl{typ: pathElement, val: 2}, pathEl{typ: pathBit, val: 1}}},
		{"11", args{"tag[41,1]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 1}}},
		{"11", args{"tag[ 41, 1]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 1}}},
		{"11", args{"tag[41 ,1]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 1}}},
		{"11", args{"tag[41 , 1]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 1}}},
		{"11", args{"tag[41,11,2]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 11}, pathEl{typ: pathElement, val: 2}}},
		{"11", args{"tag[41,11,2].x"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 11}, pathEl{typ: pathElement, val: 2}, pathEl{typ: ansiExtended, txt: "x"}}},
		{"11", args{"tag[41,11,2].1"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}, pathEl{typ: pathElement, val: 11}, pathEl{typ: pathElement, val: 2}, pathEl{typ: pathBit, val: 1}}},
		{"12", args{"tag3.count"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag3"}, pathEl{typ: ansiExtended, txt: "count"}}},
		{"13", args{"tag3[5].count"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag3"}, pathEl{typ: pathElement, val: 5}, pathEl{typ: ansiExtended, txt: "count"}}},
		{"14", args{"tag.count[712]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: ansiExtended, txt: "count"}, pathEl{typ: pathElement, val: 712}}},
		{"15", args{"tag[6].count[7]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 6}, pathEl{typ: ansiExtended, txt: "count"}, pathEl{typ: pathElement, val: 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePath(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePath(\"%s\") = %v, want %v", tt.args.p, got, tt.want)
			}
		})
	}
}

func Test_constructPath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want []uint8
	}{
		{"10", args{"tag"}, []uint8{0x91, 3, 't', 'a', 'g', 0}},
		{"11", args{"tag[41]"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 0x28, 41}},
		{"10", args{"tag.1"}, []uint8{0x91, 3, 't', 'a', 'g', 0}},
		{"11", args{"tag[41].2"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 0x28, 41}},
		{"12", args{"tag3.count"}, []uint8{0x91, 4, 't', 'a', 'g', '3', 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"13", args{"tag3[60000].count"}, []uint8{0x91, 4, 't', 'a', 'g', '3', 0x29, 0, 0x60, 0xEA, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"13", args{"tag3[70000].count"}, []uint8{0x91, 4, 't', 'a', 'g', '3', 0x2A, 0, 112, 17, 1, 0, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"14", args{"tag.count[712]"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0, 0x29, 0, 0xC8, 2}},
		{"15", args{"tag[6].count[7]"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 0x28, 6, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0, 0x28, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructPath(parsePath(tt.args.p)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructPath(\"%s\") = %v, want %v", tt.args.p, got, tt.want)
			}
		})
	}
}
