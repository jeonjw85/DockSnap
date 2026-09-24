package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jjw/docksnap/internal/store"
	"github.com/stretchr/testify/require"
)

func Test_checksumTree_hashes_symlinks_and_empty_directories(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "empty"), 0o755))
	require.NoError(t, os.Symlink("missing-a", filepath.Join(root, "link")))

	first, _, err := checksumTree(root)
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(root, "link")))
	require.NoError(t, os.Symlink("missing-b", filepath.Join(root, "link")))
	second, _, err := checksumTree(root)

	require.NoError(t, err)
	require.NotEqual(t, first, second)
}

func Test_verifySnapshot_accepts_legacy_tree_checksum(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "bind")
	id := store.VolumeID(store.VolumeBind, source, source)
	tree := filepath.Join(root, "volumes", id, "tree")
	require.NoError(t, os.MkdirAll(tree, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tree, "data"), []byte("saved"), 0o644))
	checksum, size, err := checksumTreeLegacy(tree)
	require.NoError(t, err)
	meta := store.Meta{Volumes: map[string]store.VolumeMeta{
		id: {
			Name:     source,
			Type:     store.VolumeBind,
			Source:   source,
			Bytes:    size,
			Checksum: checksum,
			Format:   store.FormatTree,
		},
	}}

	require.NoError(t, verifySnapshot(root, meta))
}

func Test_verifySnapshot_detects_changed_bind_tree(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "bind")
	id := store.VolumeID(store.VolumeBind, source, source)
	tree := filepath.Join(root, "volumes", id, "tree")
	require.NoError(t, os.MkdirAll(tree, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tree, "data"), []byte("saved"), 0o644))
	checksum, size, err := checksumTree(tree)
	require.NoError(t, err)
	meta := store.Meta{Volumes: map[string]store.VolumeMeta{
		id: {
			Name:     source,
			Type:     store.VolumeBind,
			Source:   source,
			Bytes:    size,
			Checksum: checksum,
			Format:   store.FormatTree,
		},
	}}
	require.NoError(t, verifySnapshot(root, meta))
	require.NoError(t, os.WriteFile(filepath.Join(tree, "data"), []byte("changed"), 0o644))

	require.ErrorContains(t, verifySnapshot(root, meta), "checksum mismatch")
}
