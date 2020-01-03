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
	"testing"

	"github.com/otiai10/copy"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
)

func setup() (storeDir, pluginDir string, err error) {
	tempDir, err := ioutil.TempDir("", "forensicstoreprocesstest")
	if err != nil {
		return "", "", err
	}
	storeDir = filepath.Join(tempDir, "store")
	err = os.MkdirAll(storeDir, 0755)
	if err != nil {
		return "", "", err
	}

	pluginDir = filepath.Join(tempDir, "plugins")
	err = os.MkdirAll(pluginDir, 0755)
	if err != nil {
		return "", "", err
	}

	err = copy.Copy(filepath.Join("..", "store"), storeDir)
	if err != nil {
		return "", "", err
	}
	err = copy.Copy(filepath.Join("..", "plugins"), pluginDir)
	if err != nil {
		return "", "", err
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

func Test_processJob(t *testing.T) {
	log.Println("Start setup")
	storeDir, pluginDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir, pluginDir)

	type args struct {
		jobName string
		job     Job
	}
	tests := []struct {
		name      string
		storeName string
		args      args
		wantType  string
		wantCount int
		wantErr   bool
	}{
		{"test plugin", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "example"}}, "example", 0, false},
		{"test script not existing", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "foo"}}, "", 0, true},
		// {"test docker", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "docker", Image: "alpine", Command: "true"}}, "", 0, false},

		{"test hotfixes", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "hotfixes"}}, "hotfix", 14, false},
		{"test networking", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "networking"}}, "known_network", 15, false},
		// {"test run-keys", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "run-keys"}}, "runkeys", 1, false}, TODO fix jq
		// {"test services", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "services"}}, "services", 1, false}, TODO fix services artifact
		{"test shimcache", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "shimcache"}}, "shimcache", 1024, false},
		{"test software", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "plugin", Command: "software"}}, "uninstall_entry", 133, false},
		// {"test plaso", "md1rejuc_2019-11-27T10-36-16.forensicstore", args{"testjob", Job{Type: "dockerfile", Dockerfile: "plaso"}}, "event", 123, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Workflow{
				Jobs:       map[string]Job{tt.args.jobName: tt.args.job},
				workingDir: filepath.Join(storeDir, tt.storeName),
				pluginDir:  pluginDir,
			}

			if err := w.runJob(tt.args.jobName); (err != nil) != tt.wantErr {
				t.Errorf("runJob() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				store, err := goforensicstore.NewJSONLite(filepath.Join(storeDir, tt.storeName))
				if err != nil {
					t.Fatal(err)
				}

				// validate store
				/*
					log.Println("Start validation")
					flaws, err := store.Validate()
					if err != nil {
						t.Fatal(err)
					}
					if len(flaws) > 0 {
						t.Errorf("runJob() error, validation of the forensicstore failed: %v", flaws)
					}
				*/

				log.Println("Start select")
				if tt.wantCount > 0 {
					items, err := store.Select(tt.wantType)
					if err != nil {
						t.Fatal(err)
					}
					if tt.wantCount != len(items) {
						t.Errorf("runJob() error, wrong number of resuls = %d, want %d (%v)", len(items), tt.wantCount, len(items))
					}
				}
			}
		})
	}
}
