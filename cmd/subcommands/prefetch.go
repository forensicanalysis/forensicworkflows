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
	"strings"
	"time"

	"github.com/spf13/cobra"
	"www.velocidex.com/golang/go-prefetch"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
)

func init() {
	Commands = append(Commands, Prefetch())
}

func Prefetch() *cobra.Command {
	var filtersets []string
	cmd := &cobra.Command{
		Use:   "prefetch <forensicstore>...",
		Short: "Process prefetch files",
		Args:  RequireStore,
		RunE: func(c *cobra.Command, args []string) error {
			filter := extractFilter(filtersets)

			for _, url := range args {
				store, err := goforensicstore.NewJSONLite(url)
				if err != nil {
					return err
				}
				defer store.Close()

				fileItems, err := store.Select("file", filter)
				if err != nil {
					return err
				}

				for _, item := range fileItems {
					if name, ok := item["name"]; ok {
						if name, ok := name.(string); ok {
							if strings.HasSuffix(name, ".pf") {
								if exportPath, ok := item["export_path"]; ok {
									if exportPath, ok := exportPath.(string); ok {
										file, err := store.LoadFile(exportPath)
										if err != nil {
											return err
										}

										prefetchInfo, err := prefetch.LoadPrefetch(file)
										file.Close()
										if err != nil {
											return err
										}

										_, err = store.InsertStruct(struct {
											Executable    string
											FileSize      uint32
											Hash          string
											Version       string
											LastRunTimes  []time.Time
											FilesAccessed []string
											RunCount      uint32
											Type          string
										}{
											prefetchInfo.Executable,
											prefetchInfo.FileSize,
											prefetchInfo.Hash,
											prefetchInfo.Version,
											prefetchInfo.LastRunTimes,
											prefetchInfo.FilesAccessed,
											prefetchInfo.RunCount,
											"prefetch",
										})
										if err != nil {
											return err
										}
									}
								}
							}
						}
					}
				}
			}

			return nil
		},
	}
	cmd.PersistentFlags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	return cmd
}
