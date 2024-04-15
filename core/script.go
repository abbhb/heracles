package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/shlex"
	"github.com/mrlyc/heracles/log"
	"github.com/rotisserie/eris"
)

func RunScript(ctx context.Context, command string) error {
	commands, err := shlex.Split(command)
	if err != nil {
		return eris.Wrapf(err, "failed to parse command: %s", command)
	}

	cmd := exec.CommandContext(ctx, commands[0], commands...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return eris.Wrap(err, "failed to run script")
	}

	return nil
}

type ScriptFixture struct {
	Name            string `mapstructure:"name"`
	SetupCommand    string `mapstructure:"setup"`
	TeardownCommand string `mapstructure:"teardown"`
}

func (s ScriptFixture) String() string {
	return fmt.Sprintf("ScriptFixture{Name: %s, SetupCommand: %s, TeardownCommand: %s}", s.Name, s.SetupCommand, s.TeardownCommand)
}

func (s ScriptFixture) Setup(ctx context.Context) error {
	if s.SetupCommand == "" {
		return nil
	}

	log.Infof("running setup fixture: %v", s)
	return RunScript(ctx, s.SetupCommand)
}

func (s ScriptFixture) TearDown(ctx context.Context) error {
	if s.TeardownCommand == "" {
		return nil
	}

	log.Infof("running teardown fixture: %v", s)
	return RunScript(ctx, s.TeardownCommand)
}

func NewScriptFixture(setup, teardown string) *ScriptFixture {
	return &ScriptFixture{
		SetupCommand:    setup,
		TeardownCommand: teardown,
	}
}
