// Copyright (c) 2020 Siemens AG
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// Author(s): Jonas Plum

package daggy

import (
	"reflect"
	"testing"

	"github.com/forensicanalysis/forensicstore/gostore"
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
		{"good path", args{Arguments{"foo": "bar"}, "foo"}, "bar"},
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
