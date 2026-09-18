package store

import (
	"crypto/sha256"
	"fmt"
)

func VolumeID(typ VolumeType, name, absSource string) string {
	switch typ {
	case VolumeNamed, VolumeAnonymous:
		return name
	case VolumeBind:
		sum := sha256.Sum256([]byte(absSource))
		return fmt.Sprintf("%x", sum)
	default:
		panic(fmt.Sprintf("unhandled volume type: %q", typ))
	}
}
