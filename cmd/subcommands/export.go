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
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore/goflatten"
	"github.com/forensicanalysis/forensicstore/goforensicstore"
)

func init() {
	Commands = append(Commands, Export())
}

func Export() *cobra.Command {
	var itemType string
	var filtersets []string
	outputCommand := &cobra.Command{
		Use:   "export <forensicstore>...",
		Short: "Export selected items",
		Args: func(cmd *cobra.Command, args []string) error {
			err := cmd.MarkFlagRequired("type")
			if err != nil {
				return err
			}
			return RequireStore(cmd, args)
		},
		RunE: func(rcmd *cobra.Command, args []string) error {
			filter := extractFilter(filtersets)

			for _, url := range args {
				store, err := goforensicstore.NewJSONLite(url)
				if err != nil {
					return err
				}
				defer store.Close()

				items, err := store.Select(itemType, filter)
				if err != nil {
					return err
				}
				if len(items) == 0 {
					continue
				}

				var header []string
				flatItem, err := goflatten.Flatten(items[0])
				if err != nil {
					return err
				}
				for attribute := range flatItem {
					header = append(header, attribute)
				}
				config := &outputConfig{
					Header:   header,
					Template: "", // TODO
				}
				printItem(rcmd, config, items, nil)
			}
			return nil
		},
	}
	AddOutputFlags(outputCommand)
	outputCommand.Flags().StringArrayVar(&filtersets, "filter", nil, "filter processed events")
	outputCommand.Flags().StringVar(&itemType, "type", "", "type")
	return outputCommand
}
