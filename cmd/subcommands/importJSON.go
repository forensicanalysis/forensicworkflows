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

package subcommands

import (
	"errors"
	"io/ioutil"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"

	"github.com/forensicanalysis/forensicstore"
)

func JSONImport() *cobra.Command {
	var file string
	var filtersets []string
	cmd := &cobra.Command{
		Use:   "import-json <forensicstore>...",
		Short: "Import json files",
		Args: func(cmd *cobra.Command, args []string) error {
			err := RequireStore(cmd, args)
			if err != nil {
				return err
			}
			return cmd.MarkFlagRequired("file")
		},
		RunE: func(_ *cobra.Command, args []string) error {
			filter := extractFilter(filtersets)

			for _, url := range args {
				store, teardown, err := forensicstore.Open(url)
				if err != nil {
					return err
				}
				defer teardown()

				b, err := ioutil.ReadFile(file) // #nosec
				if err != nil {
					return err
				}

				topLevel := gjson.GetBytes(b, "@this")
				if !topLevel.IsArray() {
					return errors.New("imported json must have a top level array containing objects")
				}

				topLevel.ForEach(func(_, element gjson.Result) bool {
					elementType := element.Get("type")
					if elementType.Exists() && filter.Match(forensicstore.JSONElement(element.Raw)) {
						_, err = store.Insert(forensicstore.JSONElement(element.Raw))
						if err != nil {
							return false
						}
					}
					return true
				})
			}

			return nil
		},
	}
	AddOutputFlags(cmd)
	cmd.Flags().StringVar(&file, "file", "", "forensicstore")
	cmd.Flags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	return cmd
}
