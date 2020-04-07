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
	"log"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/pkg/errors"
)

// Workflow can be used to parse workflow.yml files.
type Workflow struct {
	Tasks      map[string]Task `yaml:"tasks"`
	Arguments  Arguments       `yaml:"with"`
	graph      *dag.AcyclicGraph
	workingDir string
	pluginDir  string
	plugins    map[string]Plugin
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
func (workflow *Workflow) Run(workingDir, pluginDir string, plugins map[string]Plugin, arguments Arguments) error {
	workflow.workingDir = workingDir
	workflow.pluginDir = pluginDir
	workflow.Arguments = arguments
	workflow.plugins = plugins

	w := &dag.Walker{Callback: func(v dag.Vertex) tfdiags.Diagnostics {
		err := workflow.runTask(v.(string))
		if err != nil {
			return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, fmt.Sprint(v.(string)), err.Error())}
		}
		return nil
	}}
	w.Update(workflow.graph)
	return w.Wait().Err()
}

func (workflow *Workflow) runTask(taskName string) (err error) {
	task := workflow.Tasks[taskName]

	log.Println("Start", taskName)
	defer log.Println("End", taskName)
	switch task.Type {
	case "bash":
		return bash(task.Command, task.Arguments, task.Filter, workflow)
	case "docker":
		return docker(task.Image, task.Command, task.Arguments, task.Filter, true, workflow)
	case "dockerfile":
		return dockerfile(task.Dockerfile, task.Arguments, task.Filter, workflow)
	case "plugin":
		return plugin(task.Command, task.Arguments, task.Filter, workflow)
	default:
		return errors.New("unknown type")
	}
}
