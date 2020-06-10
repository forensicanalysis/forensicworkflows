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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/cmd/subcommands"
)

func dockerCommands() []*cobra.Command {
	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second) // TODO: adjust time
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	options := types.ImageListOptions{All: true}
	imageSummaries, err := cli.ImageList(timeoutCtx, options)
	if err != nil {
		log.Printf("docker plugins disabled: %s", err)
		return nil
	}

	var commands []*cobra.Command
	commandNames := map[string]bool{}
	for _, imageSummary := range imageSummaries {
		for _, dockerImage := range imageSummary.RepoTags {
			name, err := commandName(dockerImage)
			if err != nil {
				continue
			}

			cmd := dockerCommand(name, dockerImage, imageSummary.Labels)
			commands = append(commands, cmd)
			commandNames[name] = true
		}
	}
	for _, dockerImage := range dockerImages {
		name, err := commandName(dockerImage)
		if err != nil {
			continue
		}
		if _, ok := commandNames[name]; !ok {
			labels := map[string]string{"short": fmt.Sprintf("Use '%s install -f' to download", os.Args[0])}
			commands = append(commands, dockerCommand(name, dockerImage, labels))
		}
	}

	return commands
}

func dockerCommand(name, image string, labels map[string]string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "(docker: " + image + ")",
		Args:  subcommands.RequireStore,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Println("run", cmd.Name(), args[0])

			mounts, err := parseMounts(labels, args, cmd)
			if err != nil {
				return err
			}

			args = toCommandlineArgs(cmd.Flags(), args)
			out, err := docker(image, args, mounts)
			if err != nil {
				return err
			}

			subcommands.Print(out, cmd, args[0])
			return nil
		},
	}
	setFlags(labels, cmd)

	if use, ok := labels["use"]; ok {
		cmd.Use = use
	}
	if short, ok := labels["short"]; ok {
		cmd.Short = short + " (docker: " + image + ")"
	}

	return cmd
}

func parseMounts(labels map[string]string, args []string, cmd *cobra.Command) (map[string]string, error) {
	abs, err := filepath.Abs(args[0])
	if err != nil {
		return nil, err
	}
	mounts := map[string]string{
		abs: "input.forensicstore",
	}

	// TODO: check if application id == "eldr"
	_, err = os.Stat(strings.TrimSuffix(abs, ".forensicstore"))
	if err == nil {
		mounts[strings.TrimSuffix(abs, ".forensicstore")] = "input"
	}

	if mountsList, ok := labels["mounts"]; ok {
		for _, mountPoint := range strings.Split(mountsList, ",") {
			mountPointValue, err := cmd.Flags().GetString(mountPoint)
			if err != nil {
				continue
			}
			abs, err := filepath.Abs(mountPointValue)
			if err != nil {
				continue
			}
			mounts[abs] = mountPoint
		}
	}
	return mounts, nil
}

func setFlags(labels map[string]string, cmd *cobra.Command) {
	if use, ok := labels["arguments"]; ok {
		var schema JSONSchema
		err := json.Unmarshal([]byte(use), &schema)
		if err != nil {
			log.Println(err)
		} else {
			err := jsonschemaToFlags(schema, cmd)
			if err != nil {
				log.Println(err)
			}
		}
	}
	subcommands.AddOutputFlags(cmd)
}

func commandName(image string) (string, error) {
	idx := strings.LastIndex(image, "/")
	if strings.HasPrefix(image[idx+1:], appName+"-") {
		name := image[idx+len(appName)+2:]
		parts := strings.Split(name, ":")
		name = parts[0]
		return name, nil
	}
	return "", errors.New("no plugin")
}

func docker(image string, args []string, mountDirs map[string]string) (io.ReadCloser, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	mounts, err := getMounts(mountDirs)
	if err != nil {
		return nil, err
	}

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{Image: image, Cmd: args, Tty: true, WorkingDir: "/elementary"},
		&container.HostConfig{Binds: mounts}, // , AutoRemove: true
		nil,
		"",
	)

	if err != nil {
		return nil, err
	}

	log.Println("start docker container")
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	log.Println("wait for docker container")
	statusChannel, errChannel := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errChannel:
		if err != nil {
			return nil, err
		}
	case <-statusChannel:
	}

	log.Println("get docker container logs")
	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, err
	}

	return out, err
}

func getMounts(mountDirs map[string]string) ([]string, error) {
	for localDir := range mountDirs {
		// create directory if not exists
		_, err := os.Open(localDir) // #nosec
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s does not exist", localDir)
		} else if err != nil {
			return nil, err
		}
	}
	for localDir := range mountDirs {
		if localDir[1] == ':' {
			mountDirs["/"+strings.ToLower(string(localDir[0]))+filepath.ToSlash(localDir[2:])] = mountDirs[localDir]
			delete(mountDirs, localDir)
		}
	}

	var mounts []string
	for localDir, containerDir := range mountDirs {
		// mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: localDir, Target: "/" + containerDir})
		mounts = append(mounts, localDir+":/elementary/"+containerDir)
	}
	return mounts, nil
}
