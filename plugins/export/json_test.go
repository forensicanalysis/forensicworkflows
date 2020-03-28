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

package export

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"

	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func setup() (storeDir, pluginDir string, err error) {
	tempDir, err := ioutil.TempDir("", "forensicstoreprocesstest")
	if err != nil {
		return "", "", err
	}
	storeDir = filepath.Join(tempDir, "test")
	err = os.MkdirAll(storeDir, 0755)
	if err != nil {
		return "", "", err
	}

	pluginDir = filepath.Join(tempDir, "plugins", "imports")
	err = os.MkdirAll(pluginDir, 0755)
	if err != nil {
		return "", "", err
	}

	err = copy.Copy(filepath.Join("..", "..", "test"), storeDir)
	if err != nil {
		return "", "", err
	}
	err = copy.Copy(filepath.Join("."), pluginDir)
	if err != nil {
		return "", "", err
	}

	infos, err := ioutil.ReadDir(pluginDir)
	if err != nil {
		return "", "", err
	}
	for _, info := range infos {
		err := os.Chmod(filepath.Join(pluginDir, info.Name()), 0755)
		if err != nil {
			return "", "", err
		}
	}

	return storeDir, pluginDir, nil
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

func TestJSONPlugin_Run(t *testing.T) {
	log.Println("Start setup")
	storeDir, pluginDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir, pluginDir)

	type args struct {
		url  string
		data daggy.Arguments
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"json", args{"example1.forensicstore", daggy.Arguments{"file": filepath.Join(storeDir, "export.json")}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js := &JSONPlugin{}
			if err := js.Run(filepath.Join(storeDir, "data", tt.args.url), tt.args.data, nil); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
