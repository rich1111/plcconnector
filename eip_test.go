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
		{"09", args{"tag1234[3].ss[2].4"}, nil},
		{"10", args{"4tag"}, nil},
		{"12", args{"tag3.5count"}, nil},
		{"12", args{"tag3.count."}, nil},
		{"12", args{"tag3.count["}, nil},
		{"12", args{"tag3.count]"}, nil},
		{"17", args{"ta g1ss[2]"}, nil},
		{"18", args{"ta'g1ss[2]"}, nil},
		{"10", args{"tag"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}}},
		{"11", args{"tag[41]"}, []pathEl{pathEl{typ: ansiExtended, txt: "tag"}, pathEl{typ: pathElement, val: 41}}},
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
		{"11", args{"tag[41]"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 8, 41}},
		{"12", args{"tag3.count"}, []uint8{0x91, 4, 't', 'a', 'g', '3', 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"13", args{"tag3[60000].count"}, []uint8{0x91, 4, 't', 'a', 'g', '3', 9, 0, 0x60, 0xEA, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0}},
		{"13", args{"tag3[70000].count"}, nil},
		{"14", args{"tag.count[712]"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0, 9, 0, 0xC8, 2}},
		{"15", args{"tag[6].count[7]"}, []uint8{0x91, 3, 't', 'a', 'g', 0, 8, 6, 0x91, 5, 'c', 'o', 'u', 'n', 't', 0, 8, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructPath(parsePath(tt.args.p)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructPath(\"%s\") = %v, want %v", tt.args.p, got, tt.want)
			}
		})
	}
}
