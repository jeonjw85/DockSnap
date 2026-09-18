package main

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/jjw/docksnap/internal/store"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func Test_runListCommand_prints_exact_line_when_stdout_is_not_terminal(t *testing.T) {
	cwd := t.TempDir()
	chdir(t, cwd)
	require.NoError(t, store.New(cwd).Save(listMeta(
		t,
		"seed1",
		time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
	)))
	original := stdoutIsTerminal
	stdoutIsTerminal = func(io.Writer) bool { return false }
	t.Cleanup(func() { stdoutIsTerminal = original })
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(out)

	err := runListCommand(cmd)

	require.NoError(t, err)
	require.Equal(t, "seed1 1B 2026-09-15 14:11\n", out.String())
}
