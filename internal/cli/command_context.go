package cli

import (
	"fmt"
	"io"
)

type commandContext struct {
	stdout          io.Writer
	stderr          io.Writer
	configPath      string
	profileOverride string
	deps            Dependencies
}

type commandConfigPolicy int

const (
	commandConfigValidated commandConfigPolicy = iota + 1
	commandConfigProjectOnly
)

type commandParams struct {
	configPath      string
	profileOverride string
	configPolicy    commandConfigPolicy
	args            []string
}

func newCommandInvocation(
	deps Dependencies,
	stdout, stderr io.Writer,
	configPath, profileOverride *string,
	policy commandConfigPolicy,
	args []string,
) (commandContext, commandParams, error) {
	if runtimeDepsMissing(deps) {
		return commandContext{}, commandParams{}, runtimeError(fmt.Errorf("internal error: missing dependencies"))
	}

	return commandContext{
			stdout: stdout,
			stderr: stderr,
			deps:   deps,
		}, commandParams{
			configPath:      *configPath,
			profileOverride: *profileOverride,
			configPolicy:    policy,
			args:            args,
		}, nil
}
