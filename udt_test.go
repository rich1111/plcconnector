package plcconnector

import (
	"reflect"
	"testing"
)

const t1 = `DATATYPE POSITION
	DINT x;
	DINT y;
END_DATATYPE`

const t2 = `DATATYPE HMM (FamilyType := NoFamily)
	POSITION sprites[8];
	LINT money;
END_DATATYPE`

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
		name string
		args string
		want []T
	}{
		{"01", "INT", []T{{T: "INT"}}},
		{"01", "INT[4]", []T{{T: "INT", C: 4}}},
		{"01", "INT[4,4]", []T{{T: "INT", C: 4, C2: 4}}},
		{"01", "INT[4,4,4]", []T{{T: "INT", C: 4, C2: 4, C3: 4}}},

		{"01", t1, []T{{N: "x", T: "DINT", O: -1}, {N: "y", T: "DINT", O: -1}}},
		{"01", t2, []T{{N: "sprites", T: "POSITION", C: 8, O: -1}, {N: "money", T: "LINT", O: -1}}},
		{"01", t3, []T{{N: "x", T: "DINT", O: -1}, {N: "y", T: "DINT", O: -1}, {N: "z", T: "DINT", O: -1}}},
		{"01", t4, []T{{N: "objects", T: "POSITION3D", C: 2, O: -1}, {N: "lives", T: "SINT", O: -1}}},
		{"01", t5, []T{{N: "In", T: "BOOL", C: 0, O: 0}, {N: "Out", T: "BOOL", C: 1, O: 0}}},
		{"01", t6, []T{{N: "int", T: "INT", O: -1}, {N: "struct", T: "BOOLS", O: -1}}},
		{"01", t7, []T{{T: "INT"}}},
		{"01", t8, []T{{T: "INT"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := udtFromString(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("udtFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
