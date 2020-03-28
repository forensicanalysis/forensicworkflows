package daggy

import (
	"fmt"
	"github.com/forensicanalysis/forensicstore/gostore"
	"strings"
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
