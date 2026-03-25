package cli

import "io"

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
