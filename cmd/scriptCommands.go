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

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const appName = "forensicstore"

func scriptCommands() []*cobra.Command {
	dir, _ := os.UserConfigDir()
	scriptDir := filepath.Join(dir, "forensicstore", "scripts")
	infos, _ := ioutil.ReadDir(scriptDir)

	var commands []*cobra.Command
	for _, info := range infos {
		if info.Mode().IsRegular() && strings.HasPrefix(info.Name(), appName+"-") && !strings.HasSuffix(info.Name(), ".info") {
			commands = append(commands, scriptCommand(filepath.Join(scriptDir, info.Name())))
		}
	}

	return commands
}

func scriptCommand(path string) *cobra.Command {
	var cmd cobra.Command

	out, err := ioutil.ReadFile(path + ".info") // #nosec
	if err != nil {
		if os.IsNotExist(err) {
			// TODO: info file not exists
			log.Println(path, err)
		} else {
			log.Println(path, err)
		}
	} else {
		err = json.Unmarshal(out, &cmd)
		if err != nil {
			log.Println(err)
		}
	}

	if cmd.Use == "" {
		cmd.Use = filepath.Base(path)
	}
	cmd.Short += " (script)"
	// cmd.Args = subcommands.RequireStore
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path = filepath.ToSlash(path)
		path = strings.ReplaceAll(path, " ", `\ `)
		scriptArgs := []string{path}
		scriptArgs = append(scriptArgs, args...)
		script := exec.Command("sh", "-c", strings.Join(scriptArgs, " ")) // #nosec
		// script.Dir = url
		script.Stdout = os.Stdout
		script.Stderr = log.Writer()
		err := script.Run()
		if err != nil {
			return fmt.Errorf("%s script failed with %s", cmd.Use, err)
		}
		return nil
	}
	cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	cmd.DisableFlagParsing = true
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetHelpCommand(&cobra.Command{Use: "no-help", Hidden: true})
	return &cmd
}
