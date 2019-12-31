package plcconnector

import (
	"reflect"
	"testing"
)

const t0 = `DATATYPE POSITION DINT x;DINT y;END_DATATYPE`

const t1 = `DATATYPE POSITION
	DINT x;
	DINT y;
END_DATATYPE`

const t2 = `DATATYPE HMM (FamilyType := NoFamily)
	POSITION sprites[8];
	LINT money;
END_DATATYPE`

const t2b = `DATATYPE HMM(FamilyType:=NoFamily)POSITION sprites[8];LINT money;END_DATATYPE`

const t3 = `DATATYPE POSITION3D (FamilyType := NoFamily)
	DINT x;
	DINT y;
	DINT z;
END_DATATYPE`

const t4 = `DATATYPE MHH (FamilyType := NoFamily)
	POSITION3D objects[2];
	SINT lives;
END_DATATYPE`

const t5 = `DATATYPE BOOLS (FamilyType := NoFamily)
	BOOL In;
	BOOL Out;
END_DATATYPE`

const t6 = `DATATYPE STRINSTR (FamilyType := NoFamily)
	INT int;
	BOOLS struct;
END_DATATYPE`

const t7 = `DATATYPE UDT2 (FamilyType := NoFamily)
    DINT U2A;
    SINT U2B[3];
    UDT3 U2C (Radix := Decimal);
    UDT3 U2D[2] (Radix := Decimal);
END_DATATYPE`

const t8 = `DATATYPE MULTI (FamilyType := NoFamily)
	SINT A[3];
	SINT B[3,3];
	SINT C[3,3,3];
END_DATATYPE`

func Test_udtFromString(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		want     []udtT
		wantname string
	}{
		{"01", "INT", []udtT{{T: "INT"}}, ""},
		{"01", "INT[4]", []udtT{{T: "INT", C: 4}}, ""},
		{"01", "INT[4,4]", []udtT{{T: "INT", C: 4, C2: 4}}, ""},
		{"01", "INT[4,4,4]", []udtT{{T: "INT", C: 4, C2: 4, C3: 4}}, ""},

		{"01", t0, []udtT{{N: "x", T: "DINT", O: -1}, {N: "y", T: "DINT", O: -1}}, "POSITION"},
		{"01", t1, []udtT{{N: "x", T: "DINT", O: -1}, {N: "y", T: "DINT", O: -1}}, "POSITION"},
		{"01", t2, []udtT{{N: "sprites", T: "POSITION", C: 8, O: -1}, {N: "money", T: "LINT", O: -1}}, "HMM"},
		{"01", t2b, []udtT{{N: "sprites", T: "POSITION", C: 8, O: -1}, {N: "money", T: "LINT", O: -1}}, "HMM"},
		{"01", t3, []udtT{{N: "x", T: "DINT", O: -1}, {N: "y", T: "DINT", O: -1}, {N: "z", T: "DINT", O: -1}}, "POSITION3D"},
		{"01", t4, []udtT{{N: "objects", T: "POSITION3D", C: 2, O: -1}, {N: "lives", T: "SINT", O: -1}}, "MHH"},
		{"01", t5, []udtT{{N: "In", T: "BOOL", C: 0, O: 0}, {N: "Out", T: "BOOL", C: 1, O: 0}}, "BOOLS"},
		{"01", t6, []udtT{{N: "int", T: "INT", O: -1}, {N: "struct", T: "BOOLS", O: -1}}, "STRINSTR"},
		{"01", t7, []udtT{{N: "U2A", T: "DINT", O: -1}, {N: "U2B", T: "SINT", C: 3, O: -1}, {N: "U2C", T: "UDT3", O: -1}, {N: "U2D", T: "UDT3", C: 2, O: -1}}, "UDT2"},
		{"01", t8, []udtT{{N: "A", T: "SINT", C: 3, O: -1}, {N: "B", T: "SINT", C: 3, C2: 3, O: -1}, {N: "C", T: "SINT", C: 3, C2: 3, C3: 3, O: -1}}, "MULTI"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, gotname := udtFromString(tt.args); !reflect.DeepEqual(got, tt.want) || gotname != tt.wantname {
				t.Errorf("udtFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
