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
	"io/ioutil"
	"log"

	"github.com/hashicorp/logutils"
	"github.com/hashicorp/terraform/dag"
	"gopkg.in/yaml.v2"
)

func Parse(workflowFile string) (*Workflow, error) {
	// parse the yaml definition
	data, err := ioutil.ReadFile(workflowFile)
	if err != nil {
		return nil, err
	}
	workflow := Workflow{}
	err = yaml.Unmarshal(data, &workflow)
	if err != nil {
		return nil, err
	}

	// Create the dag
	setupLogging()
	graph := dag.AcyclicGraph{}
	jobs := map[string]Job{}
	for name, job := range workflow.Jobs {
		graph.Add(name)
		jobs[name] = job
	}

	// add edges / requirements
	for name, job := range workflow.Jobs {
		for _, requirement := range job.Requires {
			graph.Connect(dag.BasicEdge(requirement, name))
		}
	}

	workflow.graph = &graph

	return &workflow, nil
}

func setupLogging() {
	// disable logging in github.com/hashicorp/terraform/dag
	log.SetOutput(&logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"TRACE", "OTHER"},
		MinLevel: "OTHER",
		Writer:   log.Writer(),
	})
}
