package mockgen

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncDefRegexp(t *testing.T) {
	assert.True(t, funcDefRegex.MatchString("func (hi)"))
	assert.True(t, funcDefRegex.MatchString("func(hi)"))
	assert.True(t, funcDefRegex.MatchString("func(a b int) errors.Error"))
	assert.False(t, funcDefRegex.MatchString("hi func (hi)"))
	assert.False(t, funcDefRegex.MatchString("abcfunc"))
	assert.False(t, funcDefRegex.MatchString("abcfunc func(hi)"))
}

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
		}, {
			args: args{"func(a, b int) errors.Error"},
			want: []Type{
				{PackageName: "", TypeName: "func(a, b int) errors.Error", Name: ""},
			},
		}, {
			args: args{"int, func(a, b int) errors.Error"},
			want: []Type{
				{PackageName: "", TypeName: "int", Name: ""},
				{PackageName: "", TypeName: "func(a, b int) errors.Error", Name: ""},
			},
		}, {
			args: args{"d int, e func(a, b int) errors.Error"},
			want: []Type{
				{PackageName: "", TypeName: "int", Name: "d"},
				{PackageName: "", TypeName: "func(a, b int) errors.Error", Name: "e"},
			},
		}, {
			args: args{"func(int, int, errors2.Error) errors.Error"},
			want: []Type{
				{PackageName: "", TypeName: "func(int, int, errors2.Error) errors.Error", Name: ""},
			},
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, getTypesFromText(tt.args.str))
	}
}

func Test_currentTokenToParamsObjects(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			args: args{
				text: "func(a, b int) errors.Error",
			},
			want: []string{"func(a, b int) errors.Error"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := currentTokenToParamsObjects(tt.args.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("currentTokenToParamsObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseParams(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			args: args{"err1, err2 extrapkg.Error"},
			want: [][]string{
				{"err1"},
				{"err2", "extrapkg.Error"},
			},
		}, {
			args: args{"err1, err2 func(a, b int) error"},
			want: [][]string{
				{"err1"},
				{"err2", "func(a, b int) error"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseParams(tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
