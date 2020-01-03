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

// Packages forensicworkflows provides a workflow engine to automate forensic
// processes in forensicstores.
//
// Usage
//
// The command line tool requires a workflow yml file which is executed on an
// arbitrary number of forensicstores, e.g.:
//
//     forensicworkflows --workflow workflow.yml store/example1.forensicstore
//
// Workflow format
//
// The workflow.yml file contains a list of tasks like the following:
//
//     hello_task:
//         type: plugin
//         command: hello.exe
//
//     docker_task:
//         type: docker
//         image: alpine
//         command: echo bye
//         requires:
//             - hello_task
//
// There are currently 4 different types of tasks.
//
// Bash
//
// Run a script from bash. The working directory is the forensicstore. Example:
//
//     list_dir:
//         type: bash
//         command: ls
//
// Plugin
//
// Run either a builtin Go plugin or an executeable from the plugins folder. The
// working directory is the forensicstore. Example:
//
//     hotfixes:
//         type: plugin
//         command: hotfixes
//
// Docker
//
// Run a docker container. The forensicstore is located at '/store' and the plugin
// folder is located at '/plugins'. Example:
//
//     docker_task:
//         type: docker
//         image: alpine
//         command: echo bye
//
// Dockerfile
//
// Build a dockerfile from 'plugin/{dockerfile}/Dockerfile' and run the created
// image. Otherwise behaved as the docker type. Example:
//
//     dockerfalse:
//         type: dockerfile
//         dockerfile: jq
//         command: echo Dockerfile
package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/forensicanalysis/forensicworkflows/assets"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

//go:generate resources -declare -var=FS -package assets -output assets/assets.go script/* script/templates/*

func main() {
	var processCommand = &cobra.Command{
		Use:   "forensicworkflows",
		Short: "Run a workflow on the forensicstore",
		Long: `process can run parallel workflows locally. Those workflows are a directed acyclic graph of tasks. 
Those tasks can be defined to be run on the system itself or in a containerized way.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires at least one store")
			}
			for _, arg := range args {
				if _, err := os.Stat(arg); os.IsNotExist(err) {
					return errors.Wrap(os.ErrNotExist, arg)
				}
			}

			return cmd.MarkFlagRequired("workflow")
		},
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
			err = workflow.Run(storePath, "/plugins", map[string]string{
				"docker-user":     cmd.PersistentFlags().Lookup("docker-user").Value.String(),
				"docker-password": cmd.PersistentFlags().Lookup("docker-password").Value.String(),
				"docker-server":   cmd.PersistentFlags().Lookup("docker-server").Value.String(),
			})
			if err != nil {
				log.Println("processing errors: ", err)
			}
		},
	}
	processCommand.PersistentFlags().String("workflow", "", "workflow definition file")
	processCommand.PersistentFlags().String("docker-user", "", "docker username")
	processCommand.PersistentFlags().String("docker-password", "", "docker password")
	processCommand.PersistentFlags().String("docker-server", "", "docker server")
	if err := processCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
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
		if err := ioutil.WriteFile(filepath.Join(tempDir, path), content, 0644); err != nil {
			return tempDir, err
		}
		log.Printf("Unpacking %s", path)
	}

	return tempDir, nil
}
