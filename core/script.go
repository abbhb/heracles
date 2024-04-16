package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/shlex"
	"github.com/mrlyc/heracles/log"
	"github.com/rotisserie/eris"
)

type ScriptHook struct {
	Name      string   `mapstructure:"name"`
	Container string   `mapstructure:"container"`
	Setup     []string `mapstructure:"setup"`
	TearDown  []string `mapstructure:"teardown"`
}

func RunScript(ctx context.Context, command string) error {
	commands, err := shlex.Split(command)
	if err != nil {
		return eris.Wrapf(err, "failed to parse command: %s", command)
	}

	cmd := exec.CommandContext(ctx, commands[0], commands[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return eris.Wrap(err, "failed to run script")
	}

	return nil
}

func RunScripts(ctx context.Context, commands []string) error {
	for _, command := range commands {
		if err := RunScript(ctx, command); err != nil {
			return eris.Wrap(err, "failed to run script")
		}
	}

	return nil
}

type ScriptFixture struct {
	Name             string
	SetupCommands    []string
	TeardownCommands []string
}

func (s ScriptFixture) String() string {
	return fmt.Sprintf("ScriptFixture{Name: %s, SetupCommands: %v, TeardownCommands: %v}", s.Name, s.SetupCommands, s.TeardownCommands)
}

func (s ScriptFixture) Setup(ctx context.Context) error {
	if len(s.SetupCommands) == 0 {
		return nil
	}

	log.Debugf("running setup fixture: %v", s)
	return RunScripts(ctx, s.SetupCommands)
}

func (s ScriptFixture) TearDown(ctx context.Context) error {
	if len(s.TeardownCommands) == 0 {
		return nil
	}

	log.Debugf("running teardown fixture: %v", s)
	return RunScripts(ctx, s.TeardownCommands)
}

func NewScriptFixture(name string, setup, teardown []string) *ScriptFixture {
	return &ScriptFixture{
		Name:             name,
		SetupCommands:    setup,
		TeardownCommands: teardown,
	}
}

type ContainerScriptFixture struct {
	dockerCompose    *DockerCompose
	Name             string
	Container        string
	SetupCommands    []string
	TeardownCommands []string
}

func (c *ContainerScriptFixture) String() string {
	return fmt.Sprintf("ContainerScriptFixture{Name: %s, Container: %s, SetupCommands: %v, TeardownCommands: %v}", c.Name, c.Container, c.SetupCommands, c.TeardownCommands)
}

func (c *ContainerScriptFixture) runInContainer(ctx context.Context, scripts []string) error {
	container, err := c.dockerCompose.ServiceContainer(ctx, c.Container)
	if err != nil {
		return eris.Wrap(err, "failed to get container")
	}

	for _, script := range scripts {
		scripts, err := shlex.Split(script)
		if err != nil {
			return eris.Wrapf(err, "failed to parse script: %s", script)
		}

		code, reader, err := container.Exec(ctx, scripts)
		if err != nil {
			return eris.Wrapf(err, "failed to exec script: %s", script)
		}

		output := os.Stderr
		if code == 0 {
			output = os.Stdout
		}

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			_, _ = fmt.Fprintln(output, scanner.Text())
		}

		if code != 0 {
			return eris.Errorf("failed to exec script: %s, code: %d", script, code)
		}
	}

	return nil
}

func (c *ContainerScriptFixture) Setup(ctx context.Context) error {
	if len(c.SetupCommands) == 0 {
		return nil
	}

	return c.runInContainer(ctx, c.SetupCommands)
}

func (c *ContainerScriptFixture) TearDown(ctx context.Context) error {
	if len(c.TeardownCommands) == 0 {
		return nil
	}

	return c.runInContainer(ctx, c.TeardownCommands)
}

func NewContainerScriptFixture(dockerCompose *DockerCompose, name, container string, setup, teardown []string) *ContainerScriptFixture {
	return &ContainerScriptFixture{
		dockerCompose:    dockerCompose,
		Name:             name,
		Container:        container,
		SetupCommands:    setup,
		TeardownCommands: teardown,
	}
}
