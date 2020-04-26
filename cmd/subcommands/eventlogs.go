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
	"log"
	"path"
	"strings"

	"github.com/forensicanalysis/forensicworkflows/daggy"

	"github.com/Velocidex/ordereddict"
	"github.com/spf13/cobra"
	"www.velocidex.com/golang/evtx"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicstore/gostore"
)

func Eventlogs() *cobra.Command {
	var filtersets []string
	eventlogsCmd := &cobra.Command{
		Use:   "eventlogs <forensicstore>...",
		Short: "Process eventlogs into single events",
		Args:  RequireStore,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Printf("run eventlogs %s", args)
			filter := extractFilter(filtersets)

			for _, url := range args {
				err := eventlogsFromStore(url, filter, cmd)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	AddOutputFlags(eventlogsCmd)
	eventlogsCmd.Flags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	return eventlogsCmd
}

func eventlogsFromStore(url string, filter daggy.Filter, cmd *cobra.Command) error {
	store, err := goforensicstore.NewJSONLite(url)
	if err != nil {
		return err
	}
	defer store.Close()

	fileItems, err := store.Select("file", filter)
	if err != nil {
		return err
	}

	var items []gostore.Item
	for _, item := range fileItems {
		name, hasName := getString(item, "name")
		exportPath, hasExportPath := getString(item, "export_path")
		if hasName && strings.HasSuffix(name, ".evtx") && hasExportPath {
			file, err := store.Open(path.Join(url, exportPath))
			if err != nil {
				return err
			}

			events, err := getEvents(file)
			if err != nil {
				return err
			}

			items = append(items, events...)
		}
	}

	config := &outputConfig{
		Header: []string{
			"System.Computer",
			"System.TimeCreated.SystemTime",
			"System.EventRecordID",
			"System.EventID.Value",
			"System.Level",
			"System.Channel",
			"System.Provider.Name",
		},
		Template: "", // TODO
	}
	printItem(cmd, config, items, store)
	return nil
}

func getEvents(file io.ReadSeeker) ([]gostore.Item, error) {
	var items []gostore.Item

	chunks, err := evtx.GetChunks(file)
	if err != nil {
		return nil, err
	}

	for _, chunk := range chunks {
		records, err := chunk.Parse(int(chunk.Header.FirstEventRecID))
		if err != nil {
			return nil, err
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
					return nil, err
				}

				var item map[string]interface{}
				err = json.Unmarshal(serialized, &item)
				if err != nil {
					return nil, err
				}

				items = append(items, item)
			}
		}
	}

	return items, nil
}
