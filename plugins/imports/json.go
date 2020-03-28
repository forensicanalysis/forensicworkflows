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

package imports

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicstore/gojsonlite"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func init() {
	Plugins["json"] = &JSONPlugin{}
}

type JSONPlugin struct{}

func (*JSONPlugin) Description() string {
	return "Import json files"
}

func (*JSONPlugin) Run(url string, data daggy.Arguments, filter daggy.Filter) error {
	store, err := goforensicstore.NewJSONLite(url)
	if err != nil {
		return err
	}

	itemType := data.Get("type")
	if itemType == "" {
		return errors.New("missing 'type' in args")
	}

	file := data.Get("file")
	if file == "" {
		return errors.New("missing 'file' in args")
	}

	b, err := ioutil.ReadFile(file)
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

	return nil
}
