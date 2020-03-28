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
// Run either a builtin Go plugin or an executeable from the process folder. The
// working directory is the forensicstore. Example:
//
//     hotfixes:
//         type: plugin
//         command: hotfixes
//
// Docker
//
// Run a docker container. The forensicstore is located at '/store' and the plugin
// folder is located at '/process'. Example:
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
	"os"

	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/cmd"
)

//go:generate go get -u github.com/markbates/pkger/cmd/pkger
//go:generate mkdir -p assets
//go:generate pkger -o assets
//go:generate pip install -r requirements.txt

func main() {
	rootCmd := cmd.Process()
	rootCmd.AddCommand(cmd.Import(), cmd.Export())
	rootCmd.Use = "forensicworkflows"
	rootCmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
