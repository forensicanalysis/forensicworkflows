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
	"os"
	"strings"

	"github.com/forensicanalysis/forensicstore/gostore"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func RequireStore(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("the following arguments are required: forensicstore")
	}
	for _, arg := range args {
		if _, err := os.Stat(arg); os.IsNotExist(err) {
			return errors.Wrap(os.ErrNotExist, arg)
		}
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

func getString(item gostore.Item, key string) (string, bool) {
	if name, ok := item[key]; ok {
		if name, ok := name.(string); ok {
			return name, true
		}
	}
	return "", false
}
