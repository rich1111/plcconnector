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
