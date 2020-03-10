// Copyright (c) 2019 Siemens AG
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

package process

import (
	"path"
	"strings"
	"time"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicworkflows/daggy"
	"www.velocidex.com/golang/go-prefetch"
)

func init() {
	Plugins["prefetch"] = &PrefetchPlugin{}
}

type PrefetchPlugin struct{}

func (*PrefetchPlugin) Run(url string, data daggy.Data) error {
	store, err := goforensicstore.NewJSONLite(url)
	if err != nil {
		return err
	}

	fileItems, err := store.Select("file")
	if err != nil {
		return err
	}

	for _, item := range fileItems {
		if name, ok := item["name"]; ok {
			if name, ok := name.(string); ok {
				if strings.HasSuffix(name, ".pf") {
					if exportPath, ok := item["export_path"]; ok {
						if exportPath, ok := exportPath.(string); ok {
							file, err := store.Open(path.Join(url, exportPath))
							if err != nil {
								return err
							}

							prefetchInfo, err := prefetch.LoadPrefetch(file)
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

	return nil
}
