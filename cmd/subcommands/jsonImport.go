/*
 * Copyright (c) 2020 Siemens AG
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 *
 * Author(s): Jonas Plum
 */

package subcommands

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicstore/gojsonlite"
)

func init() {
	Commands = append(Commands, JSONImport())
}

func JSONImport() *cobra.Command {
	var file, itemType string
	var filtersets []string
	cmd := &cobra.Command{
		Use:   "import-json <forensicstore>...",
		Short: "Import json files",
		Args: func(cmd *cobra.Command, args []string) error {
			err := cmd.MarkFlagRequired("type")
			if err != nil {
				return err
			}
			err = RequireStore(cmd, args)
			if err != nil {
				return err
			}
			return cmd.MarkFlagRequired("file")
		},
		RunE: func(_ *cobra.Command, args []string) error {
			filter := extractFilter(filtersets)

			for _, url := range args {
				store, err := goforensicstore.NewJSONLite(url)
				if err != nil {
					return err
				}
				defer store.Close()

				b, err := ioutil.ReadFile(file) // #nosec
				if err != nil {
					return err
				}

				var items []gojsonlite.Item
				err = json.Unmarshal(b, &items)
				if err != nil {
					return errors.New("imported json must have a top level array containing objects")
				}

				for _, item := range items {
					item["type"] = itemType
					if filter.Match(item) {
						_, err = store.Insert(item)
						if err != nil {
							return err
						}
					}
				}
			}

			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&file, "file", "", "forensicstore")
	cmd.PersistentFlags().StringVar(&itemType, "type", "", "type")
	cmd.PersistentFlags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	return cmd
}
