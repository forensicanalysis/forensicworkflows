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

package cmd

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

const appName = "elementary"

func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}
	stringReader := strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}

func toCommandlineArgs(flagset *pflag.FlagSet, args []string) []string {
	var cmdArgs []string
	flagset.VisitAll(func(flag *pflag.Flag) {
		value := flag.Value.String()

		endsWithSlice := strings.HasSuffix(flag.Value.Type(), "Slice")
		if endsWithSlice && strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			slice, err := readAsCSV(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
			if err == nil {
				for _, value := range slice {
					cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%s", flag.Name, value))
				}
				return
			}
		}
		if flag.Value.Type() == "bool" {
			b, _ := flagset.GetBool(flag.Name)
			if b {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--%s", flag.Name))
			}
			return
		}
		cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%s", flag.Name, value))
	})
	for name, flags := range flagset.UnknownFlags {
		for _, flag := range flags {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%s", name, flag.Value))
		}
	}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}
