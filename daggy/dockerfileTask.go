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
