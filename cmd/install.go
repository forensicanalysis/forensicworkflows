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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicworkflows/assets"
)

// Install required assets.
func Install() *cobra.Command {
	var force bool
	var dockerUser, dockerPassword, dockerServer string
	cmd := &cobra.Command{
		Use:          "install",
		Short:        "Setup required assets",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := os.UserConfigDir()
			if err != nil {
				return err
			}

			appDir := filepath.Join(configDir, appName)

			info, err := os.Stat(appDir)
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			var auth types.AuthConfig
			auth.Username = dockerUser
			auth.Password = dockerPassword
			auth.ServerAddress = dockerServer

			if os.IsNotExist(err) {
				return setup(auth)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", appDir)
			}
			if force {
				return setup(auth)
			}
			return nil // fmt.Errorf("%s already exists, use --force to recreate", appDir)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "workflow definition file")
	cmd.Flags().StringVar(&dockerUser, "docker-user", "", "docker registry username")
	cmd.Flags().StringVar(&dockerPassword, "docker-password", "", "docker registry password")
	cmd.Flags().StringVar(&dockerServer, "docker-server", "", "docker registry server")
	return cmd
}

func setup(auth types.AuthConfig) error {
	// unpack scripts
	err := unpack()
	if err != nil {
		return err
	}

	// install python requirements
	pipPath, err := exec.LookPath("pip3")
	if err != nil {
		pipPath, err = exec.LookPath("pip")
		if err != nil {
			fmt.Println("pip is not installed")
			pipPath = ""
		}
	}
	log.Println(pipPath, "install", "-r", "requirements.txt")
	if pipPath != "" {
		pip := exec.Command(pipPath, "install", "-r", "requirements.txt") // #nosec
		err := pip.Run()
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	// pull docker images
	for _, image := range []string{} {
		log.Println("pull docker image", image)
		err = pullImage(ctx, cli, image, auth)
		if err != nil {
			return err
		}
	}

	// build docker files
	log.Println("build dockerfiles")
	return buildDockerfiles(ctx, cli, auth)
}

func unpack() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(configDir, appName)

	_ = os.RemoveAll(appDir)

	for name, data := range assets.FS {
		name = filepath.FromSlash(name)
		dest := filepath.Join(appDir, name[1:])
		log.Println("unpack", dest)
		err = os.MkdirAll(filepath.Dir(dest), 0700)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(dest, data, 0700)
		if err != nil {
			return err
		}
	}

	return err
}

func buildDockerfiles(ctx context.Context, cli *client.Client, auth types.AuthConfig) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dockerDir := filepath.Join(configDir, appName, "docker")
	infos, err := ioutil.ReadDir(dockerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, info := range infos {
		err = dockerfile(ctx, cli, info.Name(), filepath.Join(dockerDir, info.Name()), auth)
		if err != nil {
			return err
		}
	}
	return nil
}

func dockerfile(ctx context.Context, cli *client.Client, name, dir string, auth types.AuthConfig) error {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := tarFolder(dir, tw)
	if err != nil {
		return err
	}
	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	var authConfigs map[string]types.AuthConfig
	if auth.ServerAddress != "" {
		authConfigs = map[string]types.AuthConfig{
			auth.ServerAddress: auth,
		}
	}

	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
		Dockerfile:     "Dockerfile",
		Context:        dockerFileTarReader,
		Tags:           []string{appName + "-" + name},
		AuthConfigs:    authConfigs,
	}
	imageBuildResponse, err := cli.ImageBuild(ctx, dockerFileTarReader, opt)
	if err != nil {
		return errors.Wrap(err, "image build failed")
	}

	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		return errors.Wrap(err, "unable to read image build response")
	}

	return nil // docker("plugin"+dockerfile, "", arguments, filter, false, workflow)
}

func pullImage(ctx context.Context, cli *client.Client, image string, auth types.AuthConfig) error {
	body, err := cli.RegistryLogin(ctx, auth)
	if err != nil {
		return err
	}
	log.Println("login", body)

	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stderr, reader)
	if err != nil {
		return err
	}
	return nil
}
