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
	"io"
	"path"
	"strings"

	"github.com/Velocidex/ordereddict"
	"github.com/spf13/cobra"
	"www.velocidex.com/golang/evtx"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicstore/gostore"
)

func init() {
	Commands = append(Commands, Eventlogs())
}

func Eventlogs() *cobra.Command {
	var filtersets []string
	cmd := &cobra.Command{
		Use:   "eventlogs <forensicstore>...",
		Short: "Process eventlogs into single events",
		Args:  RequireStore,
		RunE: func(_ *cobra.Command, args []string) error {
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
					if name, ok := getString(item, "name"); ok {
						if strings.HasSuffix(name, ".evtx") {
							if exportPath, ok := getString(item, "export_path"); ok {
								file, err := store.Open(path.Join(url, exportPath))
								if err != nil {
									return err
								}

								err = getEvents(file, store)
								if err != nil {
									return err
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

func getString(item gostore.Item, key string) (string, bool) {
	if name, ok := item[key]; ok {
		if name, ok := name.(string); ok {
			return name, true
		}
	}
	return "", false
}

func getEvents(file io.ReadSeeker, store gostore.Store) error {
	chunks, err := evtx.GetChunks(file)
	if err != nil {
		return err
	}

	for _, chunk := range chunks {
		records, err := chunk.Parse(int(chunk.Header.FirstEventRecID))
		if err != nil {
			return err
		}

		for _, i := range records {
			eventMap, ok := i.Event.(*ordereddict.Dict)
			if ok {
				event, ok := ordereddict.GetMap(eventMap, "Event")
				if !ok {
					continue
				}

				event.Set("type", "eventlog")
				// self.maybeExpandMessage(event)

				serialized, err := json.MarshalIndent(event, " ", " ")
				if err != nil {
					return err
				}

				var item map[string]interface{}
				err = json.Unmarshal(serialized, &item)
				if err != nil {
					return err
				}

				_, err = store.Insert(item)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
