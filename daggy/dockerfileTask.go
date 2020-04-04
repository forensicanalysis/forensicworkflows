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
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

func dockerfile(dockerfile string, arguments Arguments, filter Filter, workflow *Workflow) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err = tarFolder(filepath.Join(workflow.pluginDir, dockerfile), tw)
	if err != nil {
		return err
	}
	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	var authConfigs map[string]types.AuthConfig

	var authConfig types.AuthConfig
	authConfig.Username = workflow.Arguments.Get("docker-user")
	authConfig.Password = workflow.Arguments.Get("docker-password")
	if server := workflow.Arguments.Get("docker-server"); server != "" {
		authConfig.ServerAddress = server
		authConfigs = map[string]types.AuthConfig{
			authConfig.ServerAddress: authConfig,
		}
	}

	opt := types.ImageBuildOptions{
		SuppressOutput: false,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
		Dockerfile:     "Dockerfile",
		Context:        dockerFileTarReader,
		Tags:           []string{"plugin" + dockerfile},
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

	return docker("plugin"+dockerfile, "", arguments, filter, false, workflow)
}
