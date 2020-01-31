// Copyright (c) 2019 Siemens AG
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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/forensicanalysis/forensicworkflows/plugins"
)

// Workflow can be used to parse workflow.yml files.
type Workflow struct {
	Jobs       map[string]Job `yaml:",inline"`
	graph      *dag.AcyclicGraph
	workingDir string
	pluginDir  string
	options    map[string]string
}

// A Job is a single element in a workflow.yml file.
type Job struct {
	Type       string                 `yaml:"type"`
	Requires   []string               `yaml:"requires"`
	Script     string                 `yaml:"script"`     // bash
	Image      string                 `yaml:"image"`      // docker
	Dockerfile string                 `yaml:"dockerfile"` // dockerfile
	Command    string                 `yaml:"command"`    // shared
	With       map[string]interface{} `yaml:"with"`
}

func (workflow *Workflow) Run(workingDir, pluginDir string, options map[string]string) error {
	workflow.workingDir = workingDir
	workflow.pluginDir = pluginDir
	workflow.options = options

	w := &dag.Walker{Callback: func(v dag.Vertex) tfdiags.Diagnostics {
		err := workflow.runJob(v.(string))
		if err != nil {
			return tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, fmt.Sprint(v.(string)), err.Error())}
		}
		return nil
	}}
	w.Update(workflow.graph)
	return w.Wait().Err()
}

func (workflow *Workflow) runJob(jobName string) (err error) {
	job := workflow.Jobs[jobName]

	log.Println("Start", jobName)
	defer log.Println("End", jobName)
	switch job.Type {
	case "bash":
		return workflow.bash(job.Command, job.With)
	case "docker":
		return workflow.docker(job.Image, job.Command, true, job.With)
	case "dockerfile":
		return workflow.dockerfile(job.Dockerfile, job.Command, job.With)
	case "plugin":
		return workflow.plugin(job.Command, job.With)
	default:
		return errors.New("unknown type")
	}
}

func (workflow *Workflow) bash(command string, args map[string]interface{}) (err error) {
	command = filepath.ToSlash(command)

	var stdout, stderr bytes.Buffer

	cmd := exec.Command("sh", append([]string{"-c"}, command)...)
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

func (workflow *Workflow) docker(image, command string, pull bool, args map[string]interface{}) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	var auth types.AuthConfig
	if user, ok := workflow.options["docker-user"]; ok {
		auth.Username = user
	}
	if password, ok := workflow.options["docker-password"]; ok {
		auth.Password = password
	}
	if server, ok := workflow.options["docker-server"]; ok {
		auth.ServerAddress = server
	}

	body, err := cli.RegistryLogin(ctx, auth)
	if err != nil {
		return err
	}
	fmt.Println("body", body)

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

	if workflow.workingDir[1] == ':' {
		workflow.workingDir = "/" + strings.ToLower(string(workflow.workingDir[0])) + filepath.ToSlash(workflow.workingDir[2:])
	}
	if workflow.pluginDir[1] == ':' {
		workflow.pluginDir = "/" + strings.ToLower(string(workflow.pluginDir[0])) + filepath.ToSlash(workflow.pluginDir[2:])
	}

	cmd := strings.Split(command, " ")
	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{Image: image, Cmd: cmd, Tty: true, WorkingDir: "/job"},
		&container.HostConfig{Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: workflow.workingDir, Target: "/job"},
			{Type: mount.TypeBind, Source: workflow.pluginDir, Target: "/plugins"},
		}},
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

func (workflow *Workflow) dockerfile(file, command string, args map[string]interface{}) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	fmt.Println(filepath.Join(workflow.pluginDir, file))
	infos, err := ioutil.ReadDir(filepath.Join(workflow.pluginDir, file))
	if err != nil {
		return err
	}

	for _, info := range infos {
		fmt.Println(info.Name())
		err = tarWrite(filepath.Join(workflow.pluginDir, file, info.Name()), info.Name(), tw)
		if err != nil {
			return err
		}
	}
	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	var authConfigs map[string]types.AuthConfig

	var authConfig types.AuthConfig
	if user, ok := workflow.options["docker-user"]; ok {
		authConfig.Username = user
	}
	if password, ok := workflow.options["docker-password"]; ok {
		authConfig.Password = password
	}
	if server, ok := workflow.options["docker-server"]; ok {
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
		Tags:           []string{"plugin"}, // TODO: rename tag
		AuthConfigs:    authConfigs,
	}
	imageBuildResponse, err := cli.ImageBuild(ctx, dockerFileTarReader, opt)
	if err != nil {
		log.Fatal(err)
	}

	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		log.Fatal(err, "unable to read image build response")
	}

	return workflow.docker("plugin", command, false, args) // TODO: rename tag
}

func (workflow *Workflow) plugin(command string, args map[string]interface{}) error {
	// try plugins
	if plugin, ok := plugins.Plugins[command]; ok {
		return plugin.Run(workflow.workingDir, plugins.Data{})
	}

	// try script
	parts := strings.Split(command, " ")
	cmdPath := filepath.Join(workflow.pluginDir, parts[0])
	info, err := os.Stat(cmdPath)
	if os.IsNotExist(err) {
		cmdPath = cmdPath + ".exe"
		exeinfo, err := os.Stat(cmdPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("no plugin or script `%s` found", command)
		}
		if exeinfo.IsDir() {
			return fmt.Errorf("script `%s.exe` is directory", cmdPath)
		}
	}
	if info.IsDir() {
		return fmt.Errorf("script `%s` is directory", cmdPath)
	}

	return workflow.bash(cmdPath + " " + strings.Join(parts[1:], " "), args)
}

func tarWrite(src string, dest string, tw *tar.Writer) error {
	dockerFileReader, err := os.Open(src)
	if err != nil {
		return err
	}
	readDockerFile, err := ioutil.ReadAll(dockerFileReader)
	if err != nil {
		return err
	}
	tarHeader := &tar.Header{
		Name: dest,
		Size: int64(len(readDockerFile)),
	}
	err = tw.WriteHeader(tarHeader)
	if err != nil {
		return err
	}
	_, err = tw.Write(readDockerFile)
	if err != nil {
		return err
	}
	return nil
}
