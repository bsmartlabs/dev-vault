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

func printConfigWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		if _, err := fmt.Fprintf(w, "warning: %s\n", warning); err != nil {
			return
		}
	}
}
