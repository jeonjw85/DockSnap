package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Meta_MarshalJSON_volumes_is_object_not_array(t *testing.T) {
	tag, err := NewTag("seed1")
	require.NoError(t, err)
	meta := Meta{
		Tag:       tag,
		CreatedAt: time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
		Project: ProjectMeta{
			Name:       "proj",
			WorkingDir: "/abs/path",
		},
		Volumes: map[string]VolumeMeta{
			"proj_pgdata": {
				Name:     "proj_pgdata",
				Type:     VolumeNamed,
				Source:   "proj_pgdata",
				Bytes:    12345,
				Checksum: "sha256:abcd",
				Format:   FormatTar,
			},
		},
	}

	raw, err := json.Marshal(meta)

	require.NoError(t, err)
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &decoded))
	volumes := decoded["volumes"]
	require.NotEmpty(t, volumes)
	require.Equal(t, byte('{'), volumes[0])
	require.NotEqual(t, byte('['), volumes[0])
}

func Test_Meta_MarshalJSON_tag_is_string_seed1(t *testing.T) {
	tag, err := NewTag("seed1")
	require.NoError(t, err)
	meta := Meta{
		Tag:       tag,
		CreatedAt: time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
		Project: ProjectMeta{
			Name:       "proj",
			WorkingDir: "/abs/path",
		},
		Volumes: map[string]VolumeMeta{
			"proj_pgdata": {
				Name:     "proj_pgdata",
				Type:     VolumeNamed,
				Source:   "proj_pgdata",
				Bytes:    12345,
				Checksum: "sha256:abcd",
				Format:   FormatTar,
			},
		},
	}

	raw, err := json.Marshal(meta)

	require.NoError(t, err)
	var decoded struct {
		Tag       string `json:"tag"`
		CreatedAt string `json:"createdAt"`
	}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, "seed1", decoded.Tag)
	require.Equal(t, "2026-09-15T14:11:00Z", decoded.CreatedAt)
}

func Test_Meta_MarshalJSON_contains_proj_pgdata_volume(t *testing.T) {
	tag, err := NewTag("seed1")
	require.NoError(t, err)
	meta := Meta{
		Tag:       tag,
		CreatedAt: time.Date(2026, 9, 15, 14, 11, 0, 0, time.UTC),
		Project: ProjectMeta{
			Name:       "proj",
			WorkingDir: "/abs/path",
		},
		Volumes: map[string]VolumeMeta{
			"proj_pgdata": {
				Name:     "proj_pgdata",
				Type:     VolumeNamed,
				Source:   "proj_pgdata",
				Bytes:    12345,
				Checksum: "sha256:abcd",
				Format:   FormatTar,
			},
		},
	}

	raw, err := json.Marshal(meta)

	require.NoError(t, err)
	var decoded struct {
		Project struct {
			Name       string `json:"name"`
			WorkingDir string `json:"workingDir"`
		} `json:"project"`
		Volumes map[string]VolumeMeta `json:"volumes"`
	}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, "proj", decoded.Project.Name)
	require.Equal(t, "/abs/path", decoded.Project.WorkingDir)
	vol, ok := decoded.Volumes["proj_pgdata"]
	require.True(t, ok)
	require.Equal(t, "proj_pgdata", vol.Name)
	require.Equal(t, VolumeNamed, vol.Type)
	require.Equal(t, "proj_pgdata", vol.Source)
	require.Equal(t, uint64(12345), vol.Bytes)
	require.Equal(t, "sha256:abcd", vol.Checksum)
	require.Equal(t, FormatTar, vol.Format)
}
