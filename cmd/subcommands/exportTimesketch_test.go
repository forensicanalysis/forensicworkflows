package subcommands

import (
	"github.com/forensicanalysis/forensicworkflows/daggy"
	"log"
	"path/filepath"
	"testing"

	"github.com/tidwall/gjson"

	"github.com/forensicanalysis/forensicstore"
)

func TestExportTimesketch(t *testing.T) {
	log.Println("Start setup")
	storeDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir)

	example1 := filepath.Join(storeDir, "example1.forensicstore")

	type args struct {
		url  string
		args []string
	}
	tests := []struct {
		name      string
		args      args
		wantCount int
		wantErr   bool
	}{
		{"export timesketch", args{example1, []string{}}, 4054, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := ExportTimesketch()

			command.Flags().Set("format", "none")
			command.Flags().Set("add-to-store", "true")
			command.SetArgs(append(tt.args.args, tt.args.url))
			err = command.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			store, teardown, err := forensicstore.Open(tt.args.url)
			if err != nil {
				t.Fatal(err)
			}
			defer teardown()
			elements, err := store.Select(daggy.Filter{{"type": "timesketch"}})
			if err != nil {
				t.Fatal(err)
			}

			if len(elements) != tt.wantCount {
				t.Errorf("Run() error, wrong number of resuls = %d, want %d", len(elements), tt.wantCount)
			}
		})
	}
}

func Test_jsonToText(t *testing.T) {
	type args struct {
		element gjson.Result
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"dict", args{gjson.Get(`{"a": "b"}`, "@this")}, "a: b"},
		{"list", args{gjson.Get(`["a", "b"]`, "@this")}, "a, b"},
		{"complex 1", args{gjson.Get(`{"a": ["b", "c"]}`, "@this")}, "a: b, c"},
		{"complex 2", args{gjson.Get(`{"a": ["b", "c"], "x": [1, 2]}`, "@this")}, "a: b, c; x: 1, 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jsonToText(&tt.args.element); got != tt.want {
				t.Errorf("jsonToText() = %v, want %v", got, tt.want)
			}
		})
	}
}
