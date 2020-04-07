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

package daggy

import (
	"fmt"
	"strings"

	"github.com/forensicanalysis/forensicstore/gostore"
)

// A Task is a single element in a workflow.yml file.
type Task struct {
	Type       string    `yaml:"type"`
	Requires   []string  `yaml:"requires"`
	Script     string    `yaml:"script"`     // bash
	Image      string    `yaml:"image"`      // docker
	Dockerfile string    `yaml:"dockerfile"` // dockerfile
	Command    string    `yaml:"command"`    // shared
	Arguments  Arguments `yaml:"with"`
	Filter     Filter    `yaml:"filter"`
}

// A Filter is a list of mappings that should be used for a Task.
type Filter []map[string]string

func (f Filter) toCommandline() []string {
	var cmd []string
	for _, conditions := range f {
		filterStr := ""
		for key, value := range conditions {
			if filterStr != "" {
				filterStr += ","
			}
			filterStr += key + "=" + value
		}
		cmd = append(cmd, "--filter", filterStr)
	}
	return cmd
}

// Match tests if an item matches the filter.
func (f Filter) Match(item gostore.Item) bool {
	if f == nil {
		return true
	}
	for _, condition := range f {
		if f.matchCondition(condition, item) {
			return true
		}
	}
	return false
}

func (f Filter) matchCondition(condition map[string]string, item gostore.Item) bool {
	for attribute, value := range condition {
		if !strings.Contains(fmt.Sprint(item[attribute]), value) {
			return false
		}
	}
	return true
}

// Arguments is the input into the plugins.
type Arguments map[string]string

// Get returns a single argument.
func (a Arguments) Get(name string) string {
	if value, ok := a[name]; ok {
		return value
	}
	return ""
}

func (a Arguments) toCommandline() (cmd []string) {
	for name, value := range a {
		cmd = append(cmd, "--"+name, value)
	}
	return cmd
}
