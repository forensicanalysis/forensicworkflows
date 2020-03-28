package daggy

import (
	"github.com/forensicanalysis/forensicstore/gostore"
	"reflect"
	"testing"
)

func TestArguments_Get(t *testing.T) {
	type args struct {
		data  Arguments
		field string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"good path", args{Arguments{"foo": "bar"}, "foo"}, "bar",},
		{"not existing", args{Arguments{"baz": "bar"}, "foo"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.data.Get(tt.args.field)
			if got != tt.want {
				t.Errorf("getStringField() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_toCommandline(t *testing.T) {
	tests := []struct {
		name string
		f    Filter
		want []string
	}{
		// {"simple", []map[string]string{{"file": "exe", "path": "Windows"}, {"file": "dll", "path": "Downloads"}}, []string{"--filter", "file=exe,path=Windows", "--filter", "file=dll,path=Download"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.toCommandline(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toCommandline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Match(t *testing.T) {
	type args struct {
		item gostore.Item
	}
	tests := []struct {
		name string
		f    Filter
		args args
		want bool
	}{
		{"simple match", Filter{{"name": "foo"}}, args{gostore.Item{"name": "foo"}}, true},
		{"no match", Filter{{"name": "foo"}}, args{gostore.Item{"name": "bar"}}, false},
		{"nil filter", nil, args{gostore.Item{"name": "foo"}}, true},
		{"contains match", Filter{{"name": "foo"}}, args{gostore.Item{"name": "xfool"}}, true},
		{"simple match", Filter{{"name": "foo"}}, args{gostore.Item{"name": "foo", "bar": "baz"}}, true},
		{"multi match", Filter{{"name": "foo", "bar": "baz"}}, args{gostore.Item{"name": "foo", "bar": "baz"}}, true},
		{"any match", Filter{{"x": "y"}, {"name": "foo", "bar": "baz"}}, args{gostore.Item{"name": "foo", "bar": "baz"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.Match(tt.args.item); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
