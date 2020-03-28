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
	"os"
	"path/filepath"
	"strings"
)

// Plugin is an interface that all plugins need to implement.
type Plugin interface {
	Run(store string, args Arguments, filter Filter) error
	Description() string
}

func plugin(command string, arguments Arguments, filter Filter, workflow *Workflow) error {
	// try plugins
	if plugin, ok := workflow.plugins[command]; ok {
		return plugin.Run(workflow.workingDir, arguments, filter)
	}

	// try script
	parts := strings.Split(command, " ")
	cmdPath := filepath.Join(workflow.pluginDir, parts[0])
	info, err := os.Stat(cmdPath)
	if os.IsNotExist(err) {
		cmdPath = cmdPath + ".exe"
		exeInfo, err := os.Stat(cmdPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("no plugin or script `%s` found", command)
		}
		if exeInfo.IsDir() {
			return fmt.Errorf("script `%s.exe` is directory", cmdPath)
		}
	}
	if info.IsDir() {
		return fmt.Errorf("script `%s` is directory", cmdPath)
	}

	return bash(cmdPath+" "+strings.Join(parts[1:], " "), arguments, filter, workflow)
}
