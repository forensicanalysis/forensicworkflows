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
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"text/template"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore/goflatten"
	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"github.com/forensicanalysis/forensicstore/gostore"
)

type format int

const (
	tableFormat format = iota
	csvFormat
	jsonlFormat
	reportFormat
	noneFormat
)

func (f format) string() string {
	return []string{"table", "csv", "jsonl", "report", "none"}[f]
}

func fromString(s string) format {
	for i, f := range []string{"table", "csv", "jsonl", "report", "none"} {
		if s == f {
			return format(i)
		}
	}
	return tableFormat
}

func Print(r io.Reader, cmd *cobra.Command, url string) {
	destination, format, addToStore := parseOutputFlags(cmd)

	var store *goforensicstore.ForensicStore

	if addToStore {
		var err error
		store, err = goforensicstore.NewJSONLite(url)
		if err != nil {
			store = nil
		} else {
			defer store.Close()
		}
	}
	processOutput(destination, r, format, store)
}

func printItem(cmd *cobra.Command, config *outputConfig, items []gostore.Item, store *goforensicstore.ForensicStore) {
	destination, format, addToStore := parseOutputFlags(cmd)

	if !addToStore {
		store = nil
	}
	o := newOutputWriter(destination, format, store)
	o.writeHeaderConfig(config)
	for _, item := range items {
		o.writeItem(item)
	}
	o.writeFooter()
}

func parseOutputFlags(cmd *cobra.Command) (io.Writer, format, bool) {
	var destination io.Writer = os.Stdout
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		log.Println(err)
	}
	if output != "" {
		destination, err = os.Create(output)
		if err != nil {
			log.Println(err)
		}
	}

	format := tableFormat
	formatString, err := cmd.Flags().GetString("format")
	if err != nil {
		log.Println(err)
	}
	format = fromString(formatString)

	addToStore, err := cmd.Flags().GetBool("add-to-store")
	if err != nil {
		log.Println(err)
	}

	log.Println("OUTPUT", output, format.string(), addToStore)
	return destination, format, addToStore
}

func processOutput(w io.Writer, r io.Reader, format format, store *goforensicstore.ForensicStore) {
	o := newOutputWriter(w, format, store)

	firstLine := true
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()

		// parse first line as config
		if firstLine {
			firstLine = false
			o.writeHeaderLine(line)
			continue
		}

		o.writeLine(line)
	}

	o.writeFooter()

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

type outputConfig struct {
	Header   []string `json:"header,omitempty"`
	Template string   `json:"template,omitempty"`
}

type outputWriter struct {
	format      format
	store       *goforensicstore.ForensicStore
	config      *outputConfig
	destination io.Writer

	rawOutput   bool
	tableWriter *tablewriter.Table
	csvWriter   *csv.Writer

	items []gostore.Item
}

func newOutputWriter(w io.Writer, format format, store *goforensicstore.ForensicStore) *outputWriter {
	output := &outputWriter{
		format:      format,
		store:       store,
		destination: w,
	}

	switch format {
	case csvFormat:
		output.csvWriter = csv.NewWriter(w)
	case tableFormat:
		output.tableWriter = tablewriter.NewWriter(w)
	}

	return output
}

func (o *outputWriter) writeHeaderLine(line []byte) {
	config := &outputConfig{}
	err := json.Unmarshal(line, config)
	if err != nil || reflect.DeepEqual(config, &outputConfig{}) {
		o.rawOutput = true
		log.Printf("could not unmarshal config: %s", err)
		_, err = fmt.Fprintln(o.destination, string(line))
		if err != nil {
			log.Println(err)
		}
		return
	}

	o.writeHeaderConfig(config)
}

func (o *outputWriter) writeHeaderConfig(outConfig *outputConfig) {
	o.config = outConfig

	switch o.format {
	case tableFormat:
		o.tableWriter.SetHeader(o.config.Header)
	case csvFormat:
		err := o.csvWriter.Write(o.config.Header)
		if err != nil {
			log.Println(err)
		}
	case jsonlFormat, noneFormat:
	default:
		log.Println("unknown output format:", o.format)
	}
}

func (o *outputWriter) writeLine(line []byte) {
	// just print raw output
	if o.rawOutput {
		_, err := fmt.Fprintln(o.destination, string(line))
		if err != nil {
			log.Println(err)
		}
		return
	}

	// unmarshal line
	var item gostore.Item
	err := json.Unmarshal(line, &item)
	if err != nil {
		_, err = fmt.Fprintln(o.destination, string(line))
		if err != nil {
			log.Println(err)
		}
		return
	}

	o.writeItem(item)
}
func (o *outputWriter) writeItem(item gostore.Item) {
	// add to forensicstore
	if o.store != nil {
		_, err := o.store.Insert(item)
		if err != nil {
			log.Println(err)
		}
	}

	var columns []string
	if o.format == csvFormat || o.format == tableFormat {
		flatItem, _ := goflatten.Flatten(item)

		for _, header := range o.config.Header {
			if value, ok := flatItem[header]; ok {
				columns = append(columns, fmt.Sprint(value))
			} else {
				columns = append(columns, "")
			}
		}
	}

	// print to output
	switch o.format {
	case tableFormat:
		o.tableWriter.Append(columns)
	case reportFormat:
		o.items = append(o.items, item)
	case csvFormat:
		err := o.csvWriter.Write(columns)
		if err != nil {
			log.Println(err)
		}
	case jsonlFormat:
		b, _ := json.Marshal(item)
		_, err := fmt.Fprintln(o.destination, string(b))
		if err != nil {
			log.Println(err)
		}
	}
}

func (o *outputWriter) writeFooter() {
	switch o.format {
	case csvFormat:
		o.csvWriter.Flush()
	case tableFormat:
		o.tableWriter.Render()
	case reportFormat:
		tmpl, _ := template.New("output").Parse(o.config.Template)
		_ = tmpl.Execute(o.destination, o.items)
	}

	if closer, ok := o.destination.(io.Closer); ok {
		closer.Close()
	}
}

func AddOutputFlags(command *cobra.Command) {
	command.Flags().String("output", "", "choose an output file")
	command.Flags().String("format", "table", "choose output format [csv, jsonl, table, none]")
	command.Flags().Bool("add-to-store", false, "additionally save output to store")
}
