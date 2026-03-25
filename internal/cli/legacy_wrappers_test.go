package cli

import (
	"bytes"
	"io"
)

func buildArgs(ctx commandContext, command string, argv []string) []string {
	args := []string{"dev-vault"}
	if ctx.configPath != "" {
		args = append(args, "--config", ctx.configPath)
	}
	if ctx.profileOverride != "" {
		args = append(args, "--profile", ctx.profileOverride)
	}
	args = append(args, command)
	args = append(args, argv...)
	return args
}

func runList(ctx commandContext, argv []string) int {
	return Run(buildArgs(ctx, "list", argv), ctx.stdout, ctx.stderr, ctx.deps)
}

func runPull(ctx commandContext, argv []string) int {
	return Run(buildArgs(ctx, "pull", argv), ctx.stdout, ctx.stderr, ctx.deps)
}

func runPush(ctx commandContext, argv []string) int {
	return Run(buildArgs(ctx, "push", argv), ctx.stdout, ctx.stderr, ctx.deps)
}

func printListUsage(w io.Writer) error {
	var buf bytes.Buffer
	code := Run([]string{"dev-vault", "help", "list"}, &buf, io.Discard, Dependencies{})
	if code != 0 {
		return nil
	}
	_, err := w.Write(buf.Bytes())
	return err
}
