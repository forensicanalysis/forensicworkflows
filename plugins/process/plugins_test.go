package process

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/otiai10/copy"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
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

	pluginDir = filepath.Join(tempDir, "plugins", "process")
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

func Test_plugins(t *testing.T) {
	log.Println("Start setup")
	storeDir, pluginDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir, pluginDir)

	type args struct {
		taskName string
		task     daggy.Task
	}
	tests := []struct {
		name      string
		storeName string
		args      args
		wantType  string
		wantCount int
		wantErr   bool
	}{
		{"test hotfixes", "example1.forensicstore", args{"hotfixes", daggy.Task{Type: "plugin", Command: "hotfixes"}}, "hotfix", 14, false},
		{"test networking", "example1.forensicstore", args{"networking", daggy.Task{Type: "plugin", Command: "networking"}}, "known_network", 9, false},
		{"test run-keys", "example1.forensicstore", args{"run-keys", daggy.Task{Type: "plugin", Command: "run-keys"}}, "runkey", 10, false},
		{"test services", "example1.forensicstore", args{"services", daggy.Task{Type: "plugin", Command: "services"}}, "service", 624, false},
		{"test shimcache", "example1.forensicstore", args{"shimcache", daggy.Task{Type: "plugin", Command: "shimcache"}}, "shimcache", 391, false},
		{"test software", "example1.forensicstore", args{"software", daggy.Task{Type: "plugin", Command: "software"}}, "uninstall_entry", 6, false},
		{"test prefetch", "example1.forensicstore", args{"prefetch", daggy.Task{Type: "plugin", Command: "prefetch"}}, "prefetch", 261, false},
		{"test plaso", "example1.forensicstore", args{"plaso", daggy.Task{Type: "dockerfile", Dockerfile: "plaso"}}, "event", 72, false},
		// {"test usb", "example3.forensicstore", args{"usb", daggy.Task{Type: "plugin", Command: "usb"}}, "usb-device", 6, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			defer fmt.Printf("test: %s duration %v\n", tt.name, time.Since(start))
			workflow := daggy.Workflow{Tasks: map[string]daggy.Task{tt.args.taskName: tt.args.task}}
			workflow.SetupGraph()
			if err := workflow.Run(filepath.Join(storeDir, tt.storeName), pluginDir, Plugins, nil); (err != nil) != tt.wantErr {
				t.Errorf("runTask() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				store, err := goforensicstore.NewJSONLite(filepath.Join(storeDir, tt.storeName))
				if err != nil {
					t.Fatal(err)
				}

				log.Println("Start select")
				if tt.wantCount > 0 {
					items, err := store.Select(tt.wantType)
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
