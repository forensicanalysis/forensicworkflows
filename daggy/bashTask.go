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
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"syscall"
)

func bash(command string, arguments Arguments, filter Filter, workflow *Workflow) (err error) {
	command = filepath.ToSlash(command)

	var stdout, stderr bytes.Buffer

	commandArgs := append([]string{"-c"}, command)
	commandArgs = append(commandArgs, workflow.Arguments.toCommandline()...)
	commandArgs = append(commandArgs, arguments.toCommandline()...)
	commandArgs = append(commandArgs, filter.toCommandline()...)
	cmd := exec.Command("sh", commandArgs...) // #nosec
	cmd.Dir = workflow.workingDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if waitStatus.ExitStatus() != 0 {
					return errors.New(stderr.String())
				}
			}
		} else {
			return fmt.Errorf("command `%s` failed", command)
		}
	}

	_, err = io.Copy(log.Writer(), &stdout)
	return err
}
