package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/jjw/docksnap/internal/store"
	"github.com/jjw/docksnap/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var stdoutIsTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

var pickSnapshot = tui.Pick

var lsCmd = &cobra.Command{
	Use:           "ls",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListCommand(cmd)
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

func runList() ([]store.Meta, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	list, err := store.New(cwd).List()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func runListCommand(cmd *cobra.Command) error {
	list, err := runList()
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if stdoutIsTerminal(out) && len(list) > 0 {
		tags := make([]string, len(list))
		for i, m := range list {
			tags[i] = m.Tag.String()
		}
		return pickSnapshot(os.Stdin, out, tags, func(tag string) error {
			return runRestore(cmd.Context(), tag, out)
		})
	}
	for _, m := range list {
		var n uint64
		for _, v := range m.Volumes {
			n += v.Bytes
		}
		fmt.Fprintf(out, "%s %dB %s\n", m.Tag.String(), n, m.CreatedAt.UTC().Format("2006-01-02 15:04"))
	}
	return nil
}
