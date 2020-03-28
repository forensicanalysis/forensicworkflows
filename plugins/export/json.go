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

package export

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func init() {
	Plugins["json"] = &JSONPlugin{}
}

type JSONPlugin struct{}

func (*JSONPlugin) Description() string {
	return "Export json files"
}

func (*JSONPlugin) Run(url string, data daggy.Arguments, filter daggy.Filter) error {
	store, err := goforensicstore.NewJSONLite(url)
	if err != nil {
		return err
	}

	file := data.Get("file")
	if file == "" {
		return errors.New("missing 'file' in args")
	}

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
	for _, item := range items {
		if filter.Match(item) {
			err = encoder.Encode(item)
			if err != nil {
				return err
			}
			_, err = f.WriteString(",\n")
			if err != nil {
				return err
			}
		}
	}

	_, err = f.WriteString("]\n")
	return err
}
