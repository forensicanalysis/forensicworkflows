// Copyright (c) 2019 Siemens AG
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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/otiai10/copy"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
)

func setup() (storeDir string, err error) {
	tempDir, err := ioutil.TempDir("", "forensicstoreprocesstest")
	if err != nil {
		return "", err
	}
	storeDir = filepath.Join(tempDir, "test")
	err = os.MkdirAll(storeDir, 0755)
	if err != nil {
		return "", err
	}

	err = copy.Copy(filepath.Join("..", "test"), storeDir)
	if err != nil {
		return "", err
	}

	return storeDir, nil
}

func cleanup(folders ...string) (err error) {
	for _, folder := range folders {
		err := os.RemoveAll(folder)
		if err != nil {
			return err
		}
	}
	return nil
}

func Test_processJob(t *testing.T) {
	log.Println("Start setup")
	storeDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir)

	type args struct {
		taskName string
		task     Task
	}
	tests := []struct {
		name      string
		storeName string
		args      args
		wantType  string
		wantCount int
		wantErr   bool
	}{
		{"dummy plugin", "example1.forensicstore", args{"testtask", Task{Command: "example"}}, "example", 0, false},
		{"command not existing", "example1.forensicstore", args{"testtask", Task{Command: "foo"}}, "", 0, true},
		// {"bash fail", "example1.forensicstore", args{"test bash", Task{Command: "false"}}, "", 0, true},
		// {"docker", "example1.forensicstore", args{"testtask", Task{Command: "alpine"}}, "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := Workflow{Tasks: map[string]Task{tt.args.taskName: tt.args.task}}
			workflow.SetupGraph()

			plugins := map[string]*cobra.Command{"example": &cobra.Command{
				Use: "example",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}}

			if err := workflow.Run(filepath.Join(storeDir, tt.storeName), plugins); (err != nil) != tt.wantErr {
				t.Errorf("runTask() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				store, err := goforensicstore.NewJSONLite(filepath.Join(storeDir, tt.storeName))
				if err != nil {
					t.Fatal(err)
				}

				log.Println("Start select")
				if tt.wantCount > 0 {
					items, err := store.Select(tt.wantType, nil)
					if err != nil {
						t.Fatal(err)
					}
					if tt.wantCount != len(items) {
						t.Errorf("runTask() error, wrong number of resuls = %d, want %d (%v)", len(items), tt.wantCount, len(items))
					}
				}
			}
		})
	}
}

func Test_toCmdline(t *testing.T) {
	var i interface{}
	i = []map[string]string{
		map[string]string{"foo": "bar", "bar": "baz"},
		map[string]string{"a": "b"},
	}

	type args struct {
		name string
		i    interface{}
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"filter", args{"filter", i}, []string{"--filter", "bar=baz,foo=bar", "--filter", "a=b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toCmdline(tt.args.name, tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toCmdline() = %v, want %v", got, tt.want)
			}
		})
	}
}
