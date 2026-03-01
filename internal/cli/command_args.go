package cli

import (
	"fmt"
	"strings"
)

func rejectUnexpectedArgs(parsed *parsedCommand, commandName string) error {
	if parsed == nil || parsed.fs == nil {
		return nil
	}
	extra := parsed.fs.Args()
	if len(extra) == 0 {
		return nil
	}
	return usageError(fmt.Errorf("%s does not accept positional arguments: %s", commandName, strings.Join(extra, " ")))
}
