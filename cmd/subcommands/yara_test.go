package subcommands

/*
import (
	"log"
	"path/filepath"
	"testing"

	"github.com/forensicanalysis/forensicstore"
)

func TestYara(t *testing.T) {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Start setup")
	storeDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir)

	newStorePath := filepath.Join(storeDir, "example.forensicstore")
	yaraRuleFile := filepath.Join(storeDir, "test.yar")

	store, teardown, err := forensicstore.New(newStorePath)
	if err != nil {
		t.Fatal(err)
	}

	_, f, err := store.StoreFile("test")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("my abc text here"))
	f.Close()
	err = teardown()
	if err != nil {
		t.Fatal(err)
	}

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
		{"rule file", args{newStorePath, []string{"--rules", yaraRuleFile},}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := Yara()

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
			elements, err := store.All()
			if err != nil {
				t.Fatal(err)
			}

			if len(elements) != tt.wantCount {
				t.Errorf("Run() error, wrong number of resuls = %d, want %d", len(elements), tt.wantCount)
			}
		})
	}
}
*/
