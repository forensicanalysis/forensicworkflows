// Copyright (c) 2019 Siemens AG
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
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/spf13/cobra"
)

// A Task is a single element in a workflow yml file.
type Task struct {
	Command   string                 `yaml:"command"`
	Arguments map[string]interface{} `yaml:"arguments"`
	Requires  []string               `yaml:"requires"`
}

// Workflow can be used to parse workflow yml files.
type Workflow struct {
	Tasks map[string]Task `yaml:"tasks"`
	graph *dag.AcyclicGraph
	mux   sync.Mutex
}

// SetupGraph creates a direct acyclic graph of tasks.
func (workflow *Workflow) SetupGraph() {
	// Create the dag
	setupLogging()
	graph := dag.AcyclicGraph{}
	tasks := map[string]Task{}
	for name, task := range workflow.Tasks {
		graph.Add(name)
		tasks[name] = task
	}

	// add edges / requirements
	for name, task := range workflow.Tasks {
		for _, requirement := range task.Requires {
			graph.Connect(dag.BasicEdge(requirement, name))
		}
	}

	workflow.graph = &graph
}

// Run walks the direct acyclic graph to execute each task.
func (workflow *Workflow) Run(storeDir string, plugins map[string]*cobra.Command) error {
	w := &dag.Walker{Callback: func(v dag.Vertex) tfdiags.Diagnostics {
		task := workflow.Tasks[v.(string)]

		if plugin, ok := plugins[task.Command]; ok {
			workflow.mux.Lock() // serialize tasks
			err := workflow.runTask(plugin, task, storeDir)
			workflow.mux.Unlock()
			if err != nil {
				return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, fmt.Sprint(v.(string)), err.Error())}
			}
			return nil
		}
		return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, task.Command, "command not found")}
	}}
	w.Update(workflow.graph)
	return w.Wait().Err()
}

func (workflow *Workflow) runTask(plugin *cobra.Command, task Task, storeDir string) error {
	var args []string
	for flag, value := range task.Arguments {
		args = append(args, toCmdline(flag, value)...)
	}
	args = append(args, storeDir)

	err := plugin.ParseFlags(args)
	if err != nil {
		return err
	}

	// plugin.SetArgs(args)
	if plugin.RunE == nil {
		return fmt.Errorf("plugin %s cannot be run", plugin.Name())
	}
	return plugin.RunE(plugin, plugin.Flags().Args())
}

func toCmdline(name string, i interface{}) []string {
	switch reflect.TypeOf(i).Kind() {
	case reflect.Slice:
		var s []string
		v := reflect.ValueOf(i)
		for i := 0; i < v.Len(); i++ {
			s = append(s, "--"+name, toCmdline2(v.Index(i)))
		}
		return s
	default:
		return []string{"--" + name, fmt.Sprint(i)}
	}
}

func toCmdline2(v reflect.Value) string {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice:
		var parts []string
		for i := 0; i < v.Len(); i++ {
			parts = append(parts, toCmdline2(v.Index(i)))
		}
		sort.Strings(parts)
		return strings.Join(parts, ",")
	case reflect.Map:
		var parts []string
		for _, k := range v.MapKeys() {
			i := v.MapIndex(k)
			parts = append(parts, fmt.Sprintf("%s=%s", k, i))
		}
		sort.Strings(parts)
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(v.Interface())
	}
}
