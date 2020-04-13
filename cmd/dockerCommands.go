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
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

func dockerCommands() []*cobra.Command {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	options := types.ImageListOptions{All: true}
	imageSummaries, err := cli.ImageList(ctx, options)
	if err != nil {
		return nil
	}

	var commands []*cobra.Command
	for _, imageSummary := range imageSummaries {
		for _, name := range imageSummary.RepoTags {
			idx := strings.LastIndex(name, "/")
			if strings.HasPrefix(name[idx+1:], "forensicstore-") {
				commands = append(commands, dockerCommand(name, imageSummary.Labels))
			}
		}
	}

	return commands
}

func dockerCommand(image string, labels map[string]string) *cobra.Command {
	var dockerUser, dockerPassword, dockerServer string

	name := image[14:]
	parts := strings.Split(name, ":")
	name = parts[0]

	cmd := &cobra.Command{
		Use:   name,
		Short: "(docker: " + image + ")",
		RunE: func(cmd *cobra.Command, args []string) error {
			var auth types.AuthConfig
			auth.Username = dockerUser
			auth.Password = dockerPassword
			auth.ServerAddress = dockerServer

			i := 1
			cmd.VisitParents(func(_ *cobra.Command) {
				i++
			})

			for _, url := range args {
				mounts := map[string]string{
					url: "store",
				}
				_, err := docker(image, os.Args[i:], auth, mounts)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	cmd.PersistentFlags().StringVar(&dockerUser, "docker-user", "", "docker registry username")
	cmd.PersistentFlags().StringVar(&dockerPassword, "docker-password", "", "docker registry password")
	cmd.PersistentFlags().StringVar(&dockerServer, "docker-server", "", "docker registry server")

	if use, ok := labels["use"]; ok {
		cmd.Use = use
	}
	if short, ok := labels["short"]; ok {
		cmd.Short = short + " (docker: " + image + ")"
	}

	return cmd
}

func docker(image string, args []string, auth types.AuthConfig, mountDirs map[string]string) (io.ReadCloser, error) {
	fmt.Println(image, args, mountDirs)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	for localDir := range mountDirs {
		// create directory if not exists
		_, err = os.Open(localDir) // #nosec
		if os.IsNotExist(err) {
			log.Println("creating directory", localDir)
			err = os.MkdirAll(localDir, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}

		if localDir[1] == ':' {
			mountDirs[localDir] = "/" + strings.ToLower(string(localDir[0])) + filepath.ToSlash(localDir[2:])
		}
	}

	resp, err := createContainer(ctx, cli, image, args, mountDirs)
	if err != nil {
		return nil, err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	statusChannel, errChannel := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errChannel:
		if err != nil {
			return nil, err
		}
	case <-statusChannel:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return nil, err
	}

	// stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	// _, err = ioutil.ReadAll(out)
	// _, err = io.Copy(log.Writer(), out)
	return out, err
}

func createContainer(ctx context.Context, cli *client.Client, image string, args []string, mountDirs map[string]string) (container.ContainerCreateCreatedBody, error) {
	var mounts []mount.Mount
	for localDir, containerDir := range mountDirs {
		mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: localDir, Target: "/" + containerDir})
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

	return cli.ContainerCreate(
		ctx,
		&container.Config{Image: image, Cmd: args, Tty: true, WorkingDir: "/store"},
		&container.HostConfig{Mounts: mounts}, // , AutoRemove: true
		nil,
		"",
	)
}
