package mockgen

import (
	"reflect"
	"testing"
)

func Test_getTypesFromText(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want []Type
	}{
		{
			args: args{"err1, err2 extrapkg.Error"},
			want: []Type{
				{PackageName: "extrapkg", TypeName: "Error", Name: "err1"},
				{PackageName: "extrapkg", TypeName: "Error", Name: "err2"},
			},
		}, {
			args: args{"mode, mode2 DriveMode"},
			want: []Type{
				{PackageName: "", TypeName: "DriveMode", Name: "mode"},
				{PackageName: "", TypeName: "DriveMode", Name: "mode2"},
			},
		}, {
			args: args{"DriveMode"},
			want: []Type{
				{PackageName: "", TypeName: "DriveMode", Name: ""},
			},
		}, {
			args: args{"count int"},
			want: []Type{
				{PackageName: "", TypeName: "int", Name: "count"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTypesFromText(tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getTypesFromText() = %v, want %v", got, tt.want)
			}
		})
	}
}
