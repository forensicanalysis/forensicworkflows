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
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
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
		log.Printf("could not list docker plugins: %s\n", err)
		return nil
	}

	var commands []*cobra.Command
	for _, imageSummary := range imageSummaries {
		for _, name := range imageSummary.RepoTags {
			idx := strings.LastIndex(name, "/")
			if strings.HasPrefix(name[idx+1:], appName+"-") {
				commands = append(commands, dockerCommand(name, imageSummary.Labels))
			}
		}
	}

	return commands
}

func dockerCommand(image string, labels map[string]string) *cobra.Command {
	name := image[len(appName)+1:]
	parts := strings.Split(name, ":")
	name = parts[0]

	cmd := &cobra.Command{
		Use:   name,
		Short: "(docker: " + image + ")",
		Args:  subcommands.RequireStore,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Println("run", cmd.Name(), args)

			var mountPoints []string
			if mountsList, ok := labels["mounts"]; ok {
				mountPoints = strings.Split(mountsList, ",")
			}

			for _, url := range args {
				abs, err := filepath.Abs(url)
				if err != nil {
					return err
				}
				mounts := map[string]string{
					abs: "store",
				}

				for _, mountPoint := range mountPoints {
					path, err := cmd.Flags().GetString(mountPoint)
					if err != nil {
						return err
					}
					abs, err := filepath.Abs(path)
					if err != nil {
						return err
					}
					mounts[abs] = mountPoint
				}

				args = toCommandlineArgs(cmd.Flags(), args)
				out, err := docker(image, args, mounts)
				if err != nil {
					return err
				}

				subcommands.Print(out, cmd, url)
			}
			return nil
		},
	}
	cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	subcommands.AddOutputFlags(cmd)

	if use, ok := labels["use"]; ok {
		cmd.Use = use
	}
	if short, ok := labels["short"]; ok {
		cmd.Short = short + " (docker: " + image + ")"
	}

	return cmd
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
		&container.Config{Image: image, Cmd: args, Tty: true, WorkingDir: "/store"},
		&container.HostConfig{Mounts: mounts}, // , AutoRemove: true
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

func getMounts(mountDirs map[string]string) ([]mount.Mount, error) {
	for localDir := range mountDirs {
		// create directory if not exists
		_, err := os.Open(localDir) // #nosec
		if os.IsNotExist(err) {
			log.Println("creating directory", localDir)
			err = os.MkdirAll(localDir, os.ModePerm)
			if err != nil {
				return nil, err
			}
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

	/*
		// add transit dir if import or export
		transitPath := arguments.Get("file")
		if transitPath != "" {
			transitDir, transitFile := filepath.Split(transitPath)
			if transitDir[1] == ':' {
				transitDir = "/" + strings.ToLower(string(transitDir[0])) + filepath.ToSlash(transitDir[2:])
			}
			mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: transitDir, Target: "/transit"})
			cmd = append(cmd, "--file", transitFile)
		}
	*/

	var mounts []mount.Mount
	for localDir, containerDir := range mountDirs {
		mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: localDir, Target: "/" + containerDir})
	}
	return mounts, nil
}
