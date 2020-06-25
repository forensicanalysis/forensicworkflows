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
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"

	"github.com/forensicanalysis/forensicstore"
)

type format int

const (
	tableFormat format = iota
	csvFormat
	jsonlFormat
	noneFormat
)

func fromString(s string) format {
	for i, f := range []string{"table", "csv", "jsonl", "none"} {
		if s == f {
			return format(i)
		}
	}
	return tableFormat
}

type outputConfig struct {
	Header []string `json:"header,omitempty"`
}

type OutputWriter struct {
	format      format
	store       *forensicstore.ForensicStore
	config      *outputConfig
	destination io.Writer
	firstLine   bool

	tableWriter *tablewriter.Table
	csvWriter   *csv.Writer
}

func newOutputWriter(store *forensicstore.ForensicStore, cmd *cobra.Command) *OutputWriter {
	destination, format, addToStore := parseOutputFlags(cmd)
	outStore := store
	if !addToStore {
		outStore = nil
	}

	output := &OutputWriter{
		format:      format,
		store:       outStore,
		destination: destination,
	}

	switch format {
	case csvFormat:
		output.csvWriter = csv.NewWriter(destination)
	case tableFormat:
		output.tableWriter = tablewriter.NewWriter(destination)
	}

	return output
}

func newOutputWriterStore(cmd *cobra.Command, store *forensicstore.ForensicStore, config *outputConfig) *OutputWriter {
	o := newOutputWriter(store, cmd)
	o.writeHeaderConfig(config)
	return o
}

func NewOutputWriterURL(cmd *cobra.Command, url string) (*OutputWriter, func() error) {
	var store *forensicstore.ForensicStore
	teardown := func() error { return nil }
	addToStore, err := cmd.Flags().GetBool("add-to-store")
	if err != nil && addToStore {
		var err error
		store, teardown, err = forensicstore.Open(url)
		if err != nil {
			store = nil
		}
	}
	o := newOutputWriter(store, cmd)
	o.firstLine = true
	return o, teardown
}

func (o *OutputWriter) writeHeaderLine(line []byte) {
	config := &outputConfig{}
	err := json.Unmarshal(line, config)
	if err != nil || len(config.Header) == 0 {
		log.Printf("could not unmarshal config: %s, '%s'", err, line)
		_, err = fmt.Fprintln(o.destination, string(line))
		if err != nil {
			log.Println(err)
		}
		return
	}

	o.writeHeaderConfig(config)
}

func (o *OutputWriter) writeHeaderConfig(outConfig *outputConfig) {
	o.config = outConfig
	o.firstLine = false

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

func (o *OutputWriter) Write(element []byte) (n int, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(element))
	for scanner.Scan() {
		o.writeLine(scanner.Bytes()) // nolint: errcheck
	}
	return len(element), scanner.Err()
}

func (o *OutputWriter) writeLine(element []byte) {
	if o.firstLine {
		o.writeHeaderLine(element)
		return
	}

	// print to output
	switch {
	case !gjson.ValidBytes(element) ||
		o.format == jsonlFormat ||
		(o.format == tableFormat && o.config == nil) ||
		(o.format == csvFormat && o.config == nil):
		_, err := fmt.Fprintln(o.destination, string(element))
		if err != nil {
			log.Println(err)
		}
	case o.format == tableFormat:
		o.tableWriter.Append(o.getColumns(element))
	case o.format == csvFormat:
		err := o.csvWriter.Write(o.getColumns(element))
		if err != nil {
			fmt.Fprintln(o.destination, string(element)) // nolint: errcheck
			log.Println(err)
		}
	}

	// add to forensicstore
	if o.store != nil {
		_, err := o.store.Insert(element)
		if err != nil {
			log.Println(err, string(element))
		}
	}
}

func (o *OutputWriter) getColumns(element forensicstore.JSONElement) []string {
	var columns []string
	for _, header := range o.config.Header {
		value := gjson.GetBytes(element, header)
		if value.Exists() {
			columns = append(columns, value.String())
		} else {
			columns = append(columns, "")
		}
	}
	return columns
}

func (o *OutputWriter) WriteFooter() {
	switch o.format {
	case csvFormat:
		o.csvWriter.Flush()
	case tableFormat:
		if o.tableWriter.NumLines() > 0 {
			o.tableWriter.Render()
		}
	}

	if closer, ok := o.destination.(io.Closer); ok {
		closer.Close()
	}
}

func AddOutputFlags(cmd *cobra.Command) {
	cmd.Flags().String("output", "", "choose an output file")
	cmd.Flags().String("format", "table", "choose output format [csv, jsonl, table, none]")
	cmd.Flags().Bool("add-to-store", false, "additionally save output to store")
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

	formatString, err := cmd.Flags().GetString("format")
	if err != nil {
		log.Println(err)
	}
	format := fromString(formatString)

	addToStore, err := cmd.Flags().GetBool("add-to-store")
	if err != nil {
		log.Println(err)
	}

	return destination, format, addToStore
}
