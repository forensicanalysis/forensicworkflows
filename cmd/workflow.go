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

package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/cmd/subcommands"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

// Workflow is a subcommand to run a forensic workflow.
func Workflow() *cobra.Command {
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run a workflow",
		Long: `process can run parallel workflows locally. Those workflows are a directed acyclic graph of tasks.
Those tasks can be defined to be run on the system itself or in a containerized way.`,
		Args: subcommands.RequireStore,
		RunE: func(cmd *cobra.Command, args []string) error {
			// parse workflow yaml
			workflowFile, _ := cmd.Flags().GetString("file")
			if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
				log.Fatal(err, workflowFile)
			}
			workflow, err := daggy.Parse(workflowFile)
			if err != nil {
				log.Fatal("parsing failed: ", err)
			}

			plugins := map[string]*cobra.Command{}
			for _, plugin := range allCommands() {
				plugins[plugin.Name()] = plugin
			}

			workflow.SetupGraph()
			return workflow.Run(args[0], plugins)
		},
	}
	workflowCmd.Flags().StringP("file", "f", "", "workflow definition file")
	_ = workflowCmd.MarkFlagRequired("file")
	return workflowCmd
}
