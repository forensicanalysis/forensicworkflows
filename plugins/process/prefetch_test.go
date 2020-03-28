package process

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func TestPrefetchPlugin_Run(t *testing.T) {

	type args struct {
		storeName string
		data      daggy.Arguments
		filter    daggy.Filter
	}
	tests := []struct {
		name      string
		args      args
		wantCount int
		wantErr   bool
	}{
		// {"Prefetch", args{"example1.forensicstore", nil, nil}, 261, false},
		{"Prefetch with Filter", args{"example1.forensicstore", nil, daggy.Filter{{"name": "artifactcollector%"}}}, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Println("Start setup")
			storeDir, pluginDir, err := setup()
			if err != nil {
				t.Fatal(err)
			}
			log.Println("Setup done")
			defer cleanup(storeDir, pluginDir)

			pr := &PrefetchPlugin{}

			url := filepath.Join(storeDir, "data", tt.args.storeName)
			if err := pr.Run(url, tt.args.data, tt.args.filter); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			store, err := goforensicstore.NewJSONLite(url)
			if err != nil {
				t.Errorf("goforensicstore.NewJSONLite() error = %v, wantErr %v", err, tt.wantErr)
			}
			items, err := store.Select("prefetch", nil)
			if err != nil {
				t.Errorf("store.All() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(items) != tt.wantCount {
				t.Errorf("len(items) = %v, wantCount %v", len(items), tt.wantCount)
			}

		})
	}
}
