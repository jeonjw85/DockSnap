package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jjw/docksnap/internal/store"
)

func verifySnapshot(root string, meta store.Meta) error {
	for id, volume := range meta.Volumes {
		if volume.Type != store.VolumeNamed && volume.Type != store.VolumeAnonymous && volume.Type != store.VolumeBind {
			return fmt.Errorf("%s: invalid snapshot volume type %q", volume.Name, volume.Type)
		}
		if id == "" || id == "." || !filepath.IsLocal(id) || filepath.Base(id) != id || id != store.VolumeID(volume.Type, volume.Name, volume.Source) {
			return fmt.Errorf("invalid snapshot volume id %q", id)
		}
		var path string
		var sum string
		var size uint64
		var err error
		switch volume.Format {
		case store.FormatTar:
			if volume.Type != store.VolumeNamed && volume.Type != store.VolumeAnonymous {
				return fmt.Errorf("%s: invalid snapshot format %q", volume.Name, volume.Format)
			}
			path = filepath.Join(root, "volumes", id, "data.tar")
			sum, size, err = checksumFile(path)
		case store.FormatTree:
			if volume.Type != store.VolumeBind {
				return fmt.Errorf("%s: invalid snapshot format %q", volume.Name, volume.Format)
			}
			path = filepath.Join(root, "volumes", id, "tree")
			if strings.HasPrefix(volume.Checksum, "sha256-tree-v2:") {
				sum, size, err = checksumTree(path)
			} else {
				sum, size, err = checksumTreeLegacy(path)
			}
		default:
			return fmt.Errorf("%s: invalid snapshot format %q", volume.Name, volume.Format)
		}
		if err != nil {
			return fmt.Errorf("verify snapshot volume %s: %w", volume.Name, err)
		}
		if sum != volume.Checksum || size != volume.Bytes {
			return fmt.Errorf("verify snapshot volume %s: checksum mismatch", volume.Name)
		}
	}
	return nil
}
