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
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/daggy"
	"github.com/forensicanalysis/forensicworkflows/plugins/imports"
)

func Import() *cobra.Command {
	importCommand := &cobra.Command{
		Use:   "import",
		Short: "Import data into the forensicstore",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("requires a forensicstore")
			}
			for _, arg := range args {
				if _, err := os.Stat(arg); os.IsNotExist(err) {
					return errors.Wrap(os.ErrNotExist, arg)
				}
			}
			if err := cmd.MarkFlagRequired("format"); err != nil {
				return err
			}
			return cmd.MarkFlagRequired("file")
		},
		Run: func(cmd *cobra.Command, args []string) {
			format := cmd.PersistentFlags().Lookup("format").Value.String()

			importPath := cmd.PersistentFlags().Lookup("file").Value.String()
			if _, err := os.Stat(importPath); os.IsNotExist(err) {
				log.Fatal(errors.Wrap(os.ErrNotExist, importPath))
			}
			importPath, err := filepath.Abs(importPath)
			if err != nil {
				log.Fatal(err)
			}

			workflow := &daggy.Workflow{
				Tasks: map[string]daggy.Task{
					format: {Type: "plugin", Command: format, Arguments: daggy.Arguments{"file": importPath}},
				},
			}

			arguments := getArguments(cmd)
			tasksFunc(workflow, imports.Plugins, "imports", args, arguments)
		},
	}
	importCommand.PersistentFlags().String("file", "", "imported file")
	importCommand.PersistentFlags().String("format", "", "imported format")
	importCommand.AddCommand(ListImports())
	importCommand.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	return importCommand
}

func ListImports() *cobra.Command {
	importListCommand := &cobra.Command{
		Use:   "list",
		Short: "list installed import plugins",
		Run:   listFunc(imports.Plugins, "imports"),
	}
	return importListCommand
}
