package daggy

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

func docker(image, command string, arguments Arguments, filter Filter, pull bool, workflow *Workflow) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	var auth types.AuthConfig
	auth.Username = workflow.Arguments.Get("docker-user")
	auth.Password = workflow.Arguments.Get("docker-password")
	auth.ServerAddress = workflow.Arguments.Get("docker-server")

	body, err := cli.RegistryLogin(ctx, auth)
	if err != nil {
		return err
	}
	log.Println("login", body)

	if pull {
		reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stderr, reader)
		if err != nil {
			return err
		}
	}

	// create directory if not exists
	_, err = os.Open(workflow.workingDir)
	if os.IsNotExist(err) {
		log.Println("creating directory", workflow.workingDir)
		err = os.MkdirAll(workflow.workingDir, os.ModePerm)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if workflow.workingDir[1] == ':' {
		workflow.workingDir = "/" + strings.ToLower(string(workflow.workingDir[0])) + filepath.ToSlash(workflow.workingDir[2:])
	}

	if workflow.pluginDir[1] == ':' {
		workflow.pluginDir = "/" + strings.ToLower(string(workflow.pluginDir[0])) + filepath.ToSlash(workflow.pluginDir[2:])
	}

	mounts := []mount.Mount{
		{Type: mount.TypeBind, Source: workflow.workingDir, Target: "/store"},
		{Type: mount.TypeBind, Source: workflow.pluginDir, Target: "/plugins"},
	}
	cmd := strings.Split(command, " ")
	cmd = append(cmd, workflow.Arguments.toCommandline()...) // TODO: remove "file"
	cmd = append(cmd, arguments.toCommandline()...)          // TODO: remove "file"
	cmd = append(cmd, filter.toCommandline()...)

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

	log.Printf("workingDir: %s,pluginDir: %s, cmd: %s\n", workflow.workingDir, workflow.pluginDir, cmd)
	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{Image: image, Cmd: cmd, Tty: true, WorkingDir: "/store"},
		&container.HostConfig{Mounts: mounts},
		nil,
		"",
	)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	statusChannel, errChannel := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errChannel:
		if err != nil {
			return err
		}
	case <-statusChannel:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}

	// stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	// _, err = ioutil.ReadAll(out)
	_, err = io.Copy(log.Writer(), out)
	return err
}
