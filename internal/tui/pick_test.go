package tui

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Options_builds_seed1(t *testing.T) {
	opts := Options([]string{"seed1"})
	require.Len(t, opts, 1)
	require.Equal(t, "seed1", opts[0].Value)
}

func Test_Pick_empty_tags_no_restore(t *testing.T) {
	called := false
	err := Pick(nil, nil, nil, func(string) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.False(t, called)
}

func Test_Pick_passes_selected_tag_to_restore(t *testing.T) {
	original := selectTag
	selectTag = func(io.Reader, io.Writer, []string) (string, error) {
		return "seed1", nil
	}
	t.Cleanup(func() { selectTag = original })
	var restored string

	err := Pick(bytes.NewReader(nil), io.Discard, []string{"seed1"}, func(tag string) error {
		restored = tag
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "seed1", restored)
}

func Test_Pick_does_not_restore_when_selection_is_canceled(t *testing.T) {
	original := selectTag
	selectTag = func(io.Reader, io.Writer, []string) (string, error) {
		return "", ErrCanceled
	}
	t.Cleanup(func() { selectTag = original })
	called := false

	err := Pick(bytes.NewReader(nil), io.Discard, []string{"seed1"}, func(string) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, ErrCanceled)
	require.False(t, called)
}
