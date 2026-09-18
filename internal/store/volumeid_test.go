package store

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_VolumeID_passthrough_when_named(t *testing.T) {
	name := "proj_pgdata"

	got := VolumeID(VolumeNamed, name, "")

	require.Equal(t, "proj_pgdata", got)
}

func Test_VolumeID_passthrough_when_anonymous(t *testing.T) {
	name := "abc123def"

	got := VolumeID(VolumeAnonymous, name, "")

	require.Equal(t, "abc123def", got)
}

func Test_VolumeID_is_64_hex_chars_when_bind(t *testing.T) {
	absSource := "/tmp/data"
	sum := sha256.Sum256([]byte(absSource))
	want := hex.EncodeToString(sum[:])

	got := VolumeID(VolumeBind, "data", absSource)

	require.Equal(t, 64, len(got))
	require.Equal(t, want, got)
}
