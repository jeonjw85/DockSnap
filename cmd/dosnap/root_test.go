package main

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/jjw/docksnap/internal/store"
	"github.com/stretchr/testify/require"
)

func Test_root_no_args_returns_usage_error_when_stdout_is_not_terminal(t *testing.T) {
	original := stdoutIsTerminal
	stdoutIsTerminal = func(io.Writer) bool { return false }
	t.Cleanup(func() { stdoutIsTerminal = original })
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	rootCmd.SetArgs([]string{})

	err := rootCmd.ExecuteContext(context.Background())

	require.Error(t, err)
	require.Equal(t, 2, exitStatus(err))
	require.Empty(t, out.String())
	require.Contains(t, errOut.String(), "Usage:")
}

func Test_root_no_args_runs_picker_when_stdout_is_terminal(t *testing.T) {
	cwd := t.TempDir()
	chdir(t, cwd)
	require.NoError(t, store.New(cwd).Save(listMeta(
		t,
		"seed1",
		time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
	)))
	originalTerminal := stdoutIsTerminal
	originalPicker := pickSnapshot
	stdoutIsTerminal = func(io.Writer) bool { return true }
	var picked []string
	pickSnapshot = func(_ io.Reader, _ io.Writer, tags []string, _ func(string) error) error {
		picked = append(picked, tags...)
		return nil
	}
	t.Cleanup(func() {
		stdoutIsTerminal = originalTerminal
		pickSnapshot = originalPicker
	})
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	rootCmd.SetArgs([]string{})

	err := rootCmd.ExecuteContext(context.Background())

	require.NoError(t, err)
	require.Equal(t, []string{"seed1"}, picked)
	require.Empty(t, out.String())
	require.Empty(t, errOut.String())
}
