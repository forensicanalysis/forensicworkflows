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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"

	"github.com/forensicanalysis/forensicstore"
	"github.com/forensicanalysis/forensicworkflows/daggy"
)

// Commands returns a map of all implemented commands.
func Commands() []*cobra.Command {
	return []*cobra.Command{
		Eventlogs(),
		Export(),
		ForensicStoreImport(),
		JSONImport(),
		Prefetch(),
		ImportFile(),
		// Yara(),
		ExportTimesketch(),
		BulkSearch(),
	}
}

func RequireStore(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("the following arguments are required: forensicstore")
	}
	if _, err := os.Stat(args[0]); os.IsNotExist(err) {
		return fmt.Errorf("%s: %w", args[0], os.ErrNotExist)
	}
	return nil
}

func extractFilter(filtersets []string) daggy.Filter {
	filter := daggy.Filter{}
	for _, filterset := range filtersets {
		filterelement := map[string]string{}
		for _, kv := range strings.Split(filterset, ",") {
			kvl := strings.SplitN(kv, "=", 2)
			if len(kvl) == 2 { //nolint: gomnd
				filterelement[kvl[0]] = kvl[1]
			}
		}

		filter = append(filter, filterelement)
	}
	return filter
}

func fileToReader(store *forensicstore.ForensicStore, exportPath gjson.Result) (*bytes.Reader, error) {
	file, teardown, err := store.LoadFile(exportPath.String())
	if err != nil {
		return nil, err
	}
	defer teardown()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}
