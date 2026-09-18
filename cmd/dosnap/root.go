package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/jjw/docksnap/internal/store"
	"github.com/jjw/docksnap/internal/tui"
	"github.com/spf13/cobra"
)

var newEngine = engine.New

type usageError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}

var rootCmd = &cobra.Command{
	Use:           "dosnap",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if stdoutIsTerminal(cmd.OutOrStdout()) {
			return runListCommand(cmd)
		}
		fmt.Fprint(cmd.ErrOrStderr(), cmd.UsageString())
		return &usageError{err: errors.New("command required")}
	},
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	var u *usageError
	if errors.As(err, &u) {
		return 2
	}
	if errors.Is(err, store.ErrInvalidTag) {
		return 2
	}
	if errors.Is(err, tui.ErrCanceled) {
		return 1
	}
	return 1
}
