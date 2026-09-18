package project

import "github.com/jjw/docksnap/internal/engine"

func volumesFromContainers(containers []engine.Container) []Volume {
	var out []Volume
	seen := map[string]struct{}{}
	for _, c := range containers {
		for _, m := range c.Mounts {
			v, ok := volumeFromMount(c, m)
			if !ok {
				continue
			}
			key := string(v.Type) + "\x00" + v.Name + "\x00" + v.Source
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func volumeFromMount(c engine.Container, m engine.Mount) (Volume, bool) {
	switch m.Type {
	case "tmpfs", "npipe", "image":
		return Volume{}, false
	case "bind":
		return Volume{Name: m.Source, Type: VolumeBind, Source: m.Source}, true
	case "volume":
		if anonymousMount(m) {
			if c.Labels[labelProject] == "" || c.Labels[labelService] == "" {
				return Volume{}, false
			}
			name := m.Name
			if name == "" {
				name = m.Source
			}
			return Volume{Name: name, Type: VolumeAnonymous, Source: name}, true
		}
		return Volume{Name: m.Name, Type: VolumeNamed, Source: m.Name}, true
	default:
		return Volume{}, false
	}
}

func anonymousMount(m engine.Mount) bool {
	if m.Name == "" {
		return true
	}
	if len(m.Name) != 64 {
		return false
	}
	for i := 0; i < len(m.Name); i++ {
		c := m.Name[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
