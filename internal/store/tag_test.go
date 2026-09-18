package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewTag_accepts_seed1(t *testing.T) {
	raw := "seed1"

	got, err := NewTag(raw)

	require.NoError(t, err)
	require.Equal(t, "seed1", got.String())
}

func Test_NewTag_rejects_empty(t *testing.T) {
	raw := ""

	_, err := NewTag(raw)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidTag)
	require.Contains(t, err.Error(), "tag")
}

func Test_NewTag_rejects_bad_tag_with_space(t *testing.T) {
	raw := "bad tag"

	_, err := NewTag(raw)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidTag)
	require.Contains(t, err.Error(), "tag")
}

func Test_NewTag_rejects_slash(t *testing.T) {
	raw := "a/b"

	_, err := NewTag(raw)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidTag)
	require.Contains(t, err.Error(), "tag")
}
