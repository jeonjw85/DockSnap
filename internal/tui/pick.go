package tui

import (
	"errors"
	"io"

	"github.com/charmbracelet/huh"
)

var ErrCanceled = errors.New("canceled")

var selectTag = defaultSelectTag

func Options(tags []string) []huh.Option[string] {
	opts := make([]huh.Option[string], len(tags))
	for i, tag := range tags {
		opts[i] = huh.NewOption(tag, tag)
	}
	return opts
}

func Pick(in io.Reader, out io.Writer, tags []string, restore func(string) error) error {
	if len(tags) == 0 {
		return nil
	}
	tag, err := selectTag(in, out, tags)
	if err != nil {
		return err
	}
	return restore(tag)
}

func defaultSelectTag(in io.Reader, out io.Writer, tags []string) (string, error) {
	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Restore").
				Height(12).
				Filtering(false).
				Options(Options(tags)...).
				Value(&selected),
		),
	).WithTheme(huh.ThemeBase()).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrCanceled
		}
		return "", err
	}
	return selected, nil
}
