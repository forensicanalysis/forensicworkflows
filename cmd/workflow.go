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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/daggy"
)

// Workflow is a subcommand to run a forensic worklow
func Workflow() *cobra.Command {
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run a workflow",
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
			workflowFile := cmd.Flags().Lookup("workflow").Value.String()
			if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
				log.Fatal(errors.Wrap(os.ErrNotExist, workflowFile))
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
			for _, url := range args {
				err = workflow.Run(url, plugins)
				if err != nil {
					log.Fatal("run failed: ", err)
				}
			}
		},
	}
	workflowCmd.Flags().String("workflow", "", "workflow definition file")
	return workflowCmd
}
