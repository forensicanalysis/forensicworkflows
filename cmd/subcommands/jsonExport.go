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
	"os"

	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
)

func init() {
	Commands = append(Commands, JSONExport())
}

func JSONExport() *cobra.Command {
	var file string
	var filtersets []string
	cmd := &cobra.Command{
		Use:   "export-json <forensicstore>...",
		Short: "Export json files",
		Args:  RequireStore,
		RunE: func(_ *cobra.Command, args []string) error {
			filter := extractFilter(filtersets)

			for _, url := range args {
				store, err := goforensicstore.NewJSONLite(url)
				if err != nil {
					return err
				}
				defer store.Close()

				items, err := store.All()
				if err != nil {
					return err
				}

				f, err := os.Create(file)
				if err != nil {
					return err
				}

				_, err = f.WriteString("[\n")
				if err != nil {
					return err
				}

				encoder := json.NewEncoder(f)
				first := true
				for _, item := range items {
					if filter.Match(item) {
						if !first {
							_, err = f.WriteString(",\n")
							if err != nil {
								return err
							}
						} else {
							first = false
						}
						err = encoder.Encode(item)
						if err != nil {
							return err
						}
					}
				}

				_, err = f.WriteString("]\n")
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&file, "file", "", "forensicstore")
	cmd.PersistentFlags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	return cmd
}
