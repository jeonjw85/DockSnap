package project

type VolumeType string

const (
	VolumeNamed     VolumeType = "named"
	VolumeAnonymous VolumeType = "anonymous"
	VolumeBind      VolumeType = "bind"
)

type Volume struct {
	Name   string
	Type   VolumeType
	Source string
}

type Project struct {
	Name       string
	WorkingDir string
	Containers []string
	Volumes    []Volume
}
