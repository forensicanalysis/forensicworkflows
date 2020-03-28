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
	cmd := exec.Command("sh", commandArgs...)
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
