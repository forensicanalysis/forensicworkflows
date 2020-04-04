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

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markbates/pkger"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/daggy"
)

func tasksFunc(workflow *daggy.Workflow, plugins map[string]daggy.Plugin, processDir string, stores []string, arguments daggy.Arguments) {
	workflow.SetupGraph()

	// unpack scripts
	scriptDir, err := unpack()
	if err != nil {
		log.Fatal("unpacking error: ", err)
	}
	defer os.RemoveAll(scriptDir)

	for _, store := range stores {
		// get store path
		storePath, err := filepath.Abs(store)
		if err != nil {
			log.Println("abs: ", err)
		}

		// run workflow
		err = workflow.Run(storePath, path.Join(scriptDir, processDir), plugins, arguments)
		if err != nil {
			log.Println("processing errors: ", err)
		}
	}
}

type PluginJSON struct {
	Description string
}

func listFunc(plugins map[string]daggy.Plugin, subScriptDir string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		scriptDir, err := unpack()
		if err != nil {
			log.Fatal(err)
		}
		infos, err := ioutil.ReadDir(filepath.Join(scriptDir, subScriptDir))
		if err != nil {
			log.Fatal(err)
		}

		var names []string
		list := map[string]string{}

		// get script plugins
		for _, info := range infos {
			description := ""
			b, err := ioutil.ReadFile(filepath.Join(scriptDir, subScriptDir, info.Name(), "plugin.json"))
			if err == nil {
				pluginJSON := &PluginJSON{}
				err = json.Unmarshal(b, pluginJSON)
				if err == nil {
					description = pluginJSON.Description
				}
			}

			names = append(names, info.Name())
			list[info.Name()] = description
		}

		// get internal plugins
		for name, plugin := range plugins {
			names = append(names, name)
			list[name] = plugin.Description()
		}

		// print plugins
		sort.Strings(names)
		for _, name := range names {
			description := list[name]
			if description != "" {
				name += ":"
			}
			fmt.Printf("%-20s %s\n", name, description)
		}
	}
}

func getArguments(cmd *cobra.Command) daggy.Arguments {
	arguments := daggy.Arguments{}
	for name, unknownFlags := range cmd.Flags().UnknownFlags {
		for _, unknownFlag := range unknownFlags {
			arguments[name] = unknownFlag.Value
		}
	}
	return arguments
}

func unpack() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return cacheDir, err
	}

	forensicstoreDir := filepath.Join(cacheDir, "forensicstore")
	scriptsDir := filepath.Join(forensicstoreDir, "scripts")

	_ = os.RemoveAll(scriptsDir)

	log.Printf("unpack to %s\n", forensicstoreDir)

	err = pkger.Walk("/scripts", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		parts := strings.SplitN(path, ":", 2)
		if len(parts) != 2 {
			return errors.New("could not split path")
		}

		if info.IsDir() {
			return os.MkdirAll(filepath.Join(forensicstoreDir, parts[1]), 0700)
		}

		// Copy file
		err = os.MkdirAll(filepath.Join(forensicstoreDir, filepath.Dir(parts[1])), 0700)
		if err != nil {
			return err
		}
		srcFile, err := pkger.Open(parts[1])
		if err != nil {
			return err
		}
		dstFile, err := os.Create(filepath.Join(forensicstoreDir, parts[1]))
		if err != nil {
			return err
		}
		_, err = io.Copy(dstFile, srcFile)
		return err
	})

	return scriptsDir, err
}
