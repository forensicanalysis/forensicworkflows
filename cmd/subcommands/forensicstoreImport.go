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
	"io"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicstore/gojsonlite"
	"github.com/forensicanalysis/forensicstore/gostore"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func ForensicStoreImport() *cobra.Command {
	var file string
	var filtersets []string
	cmd := &cobra.Command{
		Use:   "import-forensicstore <forensicstore>...",
		Short: "Import forensicstore files",
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
				store, err := goforensicstore.NewJSONLite(url)
				if err != nil {
					return err
				}
				defer store.Close()

				err = jsonLite(store, file, filter)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	AddOutputFlags(cmd)
	cmd.Flags().StringVar(&file, "file", "", "forensicstore")
	cmd.Flags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	return cmd
}

// jsonLite merges another JSONLite into this one.
func jsonLite(db gostore.Store, url string, filter daggy.Filter) (err error) {
	// TODO: import items with "_path" on sublevel"…
	// TODO: import does not need to unflatten and flatten

	importStore, err := gojsonlite.New(url)
	if err != nil {
		return err
	}
	items, err := importStore.All()
	if err != nil {
		return err
	}

	for _, item := range items {
		if !filter.Match(item) {
			continue
		}

		for field := range item {
			item := item
			if strings.HasSuffix(field, "_path") {
				dstPath, writer, err := db.StoreFile(item[field].(string))
				if err != nil {
					return err
				}
				reader, err := importStore.Open(filepath.Join(url, item[field].(string)))
				if err != nil {
					return err
				}
				_, err = io.Copy(writer, reader)
				reader.Close()
				writer.Close()
				if err != nil {
					return err
				}
				if err := mergo.Merge(&item, gojsonlite.Item{field: dstPath}); err != nil {
					return err
				}
			}
		}
		_, err = db.Insert(item)
		if err != nil {
			return err
		}
	}
	return err
}
