package cli

import "errors"

type commandErrorKind int

const (
	commandErrorUsage commandErrorKind = iota + 1
	commandErrorRuntime
	commandErrorOutput
)

type commandError struct {
	kind commandErrorKind
	err  error
}

func (e *commandError) Error() string {
	return e.err.Error()
}

func (e *commandError) Unwrap() error {
	return e.err
}

func wrapCommandError(kind commandErrorKind, err error) error {
	if err == nil {
		return nil
	}
	var existing *commandError
	if errors.As(err, &existing) {
		return err
	}
	return &commandError{kind: kind, err: err}
}

func usageError(err error) error {
	return wrapCommandError(commandErrorUsage, err)
}

func runtimeError(err error) error {
	return wrapCommandError(commandErrorRuntime, err)
}

func outputError(err error) error {
	return wrapCommandError(commandErrorOutput, err)
}

func exitCodeForError(err error) int {
	if err == nil {
		return 0
	}
	var commandErr *commandError
	if errors.As(err, &commandErr) {
		switch commandErr.kind {
		case commandErrorUsage:
			return 2
		default:
			return 1
		}
	}
	return 1
}
