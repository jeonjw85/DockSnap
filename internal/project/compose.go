package project

import (
	"context"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

type composeProject struct {
	Name    string
	Volumes []types.ServiceVolumeConfig
	Named   types.Volumes
}

func loadCompose(ctx context.Context, cwd, name string) (composeProject, error) {
	fns := []cli.ProjectOptionsFn{
		cli.WithWorkingDirectory(cwd),
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithInterpolation(true),
		cli.WithResolvedPaths(true),
		cli.WithConfigFileEnv,
		cli.WithDefaultConfigPath,
	}
	if name != "" {
		fns = append(fns, cli.WithName(name))
	}
	opts, err := cli.NewProjectOptions(nil, fns...)
	if err != nil {
		return composeProject{}, err
	}
	proj, err := opts.LoadProject(ctx)
	if err != nil {
		return composeProject{}, err
	}
	var vols []types.ServiceVolumeConfig
	for _, svc := range proj.Services {
		vols = append(vols, svc.Volumes...)
	}
	return composeProject{Name: proj.Name, Volumes: vols, Named: proj.Volumes}, nil
}

func volumesFromCompose(proj composeProject) []Volume {
	var out []Volume
	seen := map[string]struct{}{}
	for _, m := range proj.Volumes {
		v, ok := volumeFromComposeMount(proj, m)
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
	return out
}

func volumeFromComposeMount(proj composeProject, m types.ServiceVolumeConfig) (Volume, bool) {
	switch m.Type {
	case types.VolumeTypeTmpfs, types.VolumeTypeNamedPipe, types.VolumeTypeCluster:
		return Volume{}, false
	case types.VolumeTypeBind:
		return Volume{Name: m.Source, Type: VolumeBind, Source: m.Source}, true
	case types.VolumeTypeVolume:
		if m.Source == "" {
			return Volume{Name: m.Target, Type: VolumeAnonymous, Source: m.Target}, true
		}
		name := namedComposeVolume(proj, m.Source)
		return Volume{Name: name, Type: VolumeNamed, Source: name}, true
	default:
		return Volume{}, false
	}
}

func namedComposeVolume(proj composeProject, source string) string {
	if vol, ok := proj.Named[source]; ok {
		if vol.Name != "" {
			return vol.Name
		}
		if proj.Name != "" {
			return proj.Name + "_" + source
		}
	}
	return source
}
