package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicworkflows/assets"
	"github.com/forensicanalysis/forensicworkflows/daggy"
	"github.com/forensicanalysis/forensicworkflows/plugins/process"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func Process() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process",
		Short: "Run a workflow on the forensicstore",
		Long: `process can run parallel workflows locally. Those workflows are a directed acyclic graph of tasks.
Those tasks can be defined to be run on the system itself or in a containerized way.`,
		Args: requiredArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// parse workflow yaml
			workflowFile := cmd.PersistentFlags().Lookup("workflow").Value.String()
			if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
				log.Fatal(errors.Wrap(os.ErrNotExist, workflowFile))
			}
			workflow, err := daggy.Parse(workflowFile)
			if err != nil {
				log.Fatal("parsing failed: ", err)
			}

			workflow.SetupGraph()

			// unpack scripts
			tempDir, err := unpack()
			if err != nil {
				log.Fatal("unpacking error: ", err)
			}
			defer os.RemoveAll(tempDir)

			// get store path
			storePath, err := filepath.Abs(args[0])
			if err != nil {
				log.Println("abs: ", err)
			}

			// run workflow
			err = workflow.Run(storePath, path.Join(tempDir, "process"), process.Plugins, map[string]string{
				"docker-user":     cmd.PersistentFlags().Lookup("docker-user").Value.String(),
				"docker-password": cmd.PersistentFlags().Lookup("docker-password").Value.String(),
				"docker-server":   cmd.PersistentFlags().Lookup("docker-server").Value.String(),
			})
			if err != nil {
				log.Println("processing errors: ", err)
			}
		},
	}
	cmd.PersistentFlags().String("workflow", "", "workflow definition file")
	cmd.PersistentFlags().String("docker-user", "", "docker username")
	cmd.PersistentFlags().String("docker-password", "", "docker password")
	cmd.PersistentFlags().String("docker-server", "", "docker server")
	return cmd
}

func Import() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import another jsonlite file",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := cmd.Flags().Args()[0]
			storeName := cmd.Flags().Args()[1]
			store, err := goforensicstore.NewJSONLite(storeName)
			if err != nil {
				fmt.Println(err)
				return err
			}

			if err = store.ImportJSONLite(url); err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		},
	}
}

func unpack() (tempDir string, err error) {
	tempDir, err = ioutil.TempDir("", "forensicreports")
	if err != nil {
		return tempDir, err
	}

	for path, content := range assets.FS.Files {
		if err := os.MkdirAll(filepath.Join(tempDir, filepath.Dir(path)), 0700); err != nil {
			return tempDir, err
		}
		if err := ioutil.WriteFile(filepath.Join(tempDir, path), content, 0755); err != nil {
			return tempDir, err
		}
		log.Printf("Unpacking %s", path)
	}

	return tempDir, nil
}

func requiredArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("requires at least one store")
	}
	for _, arg := range args {
		if _, err := os.Stat(arg); os.IsNotExist(err) {
			return errors.Wrap(os.ErrNotExist, arg)
		}
	}

	return cmd.MarkFlagRequired("workflow")
}
