package store

import "time"

type VolumeType string

const (
	VolumeNamed     VolumeType = "named"
	VolumeAnonymous VolumeType = "anonymous"
	VolumeBind      VolumeType = "bind"
)

type Format string

const (
	FormatTar  Format = "tar"
	FormatTree Format = "tree"
)

type ProjectMeta struct {
	Name       string `json:"name"`
	WorkingDir string `json:"workingDir"`
}

type VolumeMeta struct {
	Name     string     `json:"name"`
	Type     VolumeType `json:"type"`
	Source   string     `json:"source"`
	Bytes    uint64     `json:"bytes"`
	Checksum string     `json:"checksum"`
	Format   Format     `json:"format"`
}

type Meta struct {
	Tag       Tag                   `json:"tag"`
	CreatedAt time.Time             `json:"createdAt"`
	Project   ProjectMeta           `json:"project"`
	Volumes   map[string]VolumeMeta `json:"volumes"`
}
