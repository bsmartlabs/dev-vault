package cli

import "io"

type commandContext struct {
	stdout          io.Writer
	stderr          io.Writer
	configPath      string
	profileOverride string
	deps            Dependencies
}
