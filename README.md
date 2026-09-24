# DockSnap

[한국어](README.ko.md)

DockSnap is a fast seed/reset tool for Docker Compose development volumes. The
CLI binary is `dosnap`. It feels like `git stash` for local Docker state: save a
named snapshot, change the data, and restore the snapshot when you want a clean
starting point.

DockSnap is intended for development workflows. It is not a production backup
system.

## Features

- Saves named volumes, project-attached anonymous volumes, and bind mounts.
- Limits every operation to the Compose project associated with the current directory.
- Discovers running projects through Docker Engine labels, with a Compose file fallback.
- Pauses running project containers while saving.
- Stops running containers before restore and restarts only those that were running.
- Provides an interactive snapshot picker when stdout is a terminal.
- Uses daemon-free unit tests backed by a fake Engine implementation.

## Requirements

- Go 1.23 or later to build from source.
- A reachable Docker Engine.
- A Docker Compose project in the current directory.
- Optional: `rsync` for bind-mount copies. DockSnap uses a Go fallback when it is unavailable.

DockSnap respects `DOCKER_HOST` and defaults to `unix:///var/run/docker.sock`.

## Build From Source

```bash
go build -o bin/dosnap ./cmd/dosnap
```

Place `bin/dosnap` on your `PATH` if you want to invoke it as `dosnap`.

## Quick Start

Run these commands from the Compose project directory:

```bash
dosnap save seed1
dosnap ls
dosnap restore seed1
```

Restore speed depends on the amount of data and the host storage.

## Commands

| Command | Behavior |
| --- | --- |
| `dosnap save <tag>` | Resolve the project, pause its running containers, copy its volumes, write metadata, and attempt to unpause every container DockSnap paused. |
| `dosnap restore <tag>` | Validate the project, stop its running containers, overwrite the snapshot volumes, and restart the containers that were running. |
| `dosnap ls` | List snapshots and sizes. When stdout is a terminal and snapshots exist, open the restore picker. |
| `dosnap` | Open the restore picker when stdout is a terminal; otherwise print usage and exit with status 2. |

Tags must match `^[A-Za-z0-9._-]+$`.

## Snapshot Scope and Storage

DockSnap snapshots the Compose project of the current directory, never the
entire Docker daemon. It includes named volumes, project-attached anonymous
volumes, and bind mounts. It skips unsupported mount types such as `tmpfs`,
named pipes, and image mounts.

Snapshots are stored under:

```text
.dosnap/snapshots/<tag>/
  meta.json
  volumes/<volume-id>/data.tar
  volumes/<volume-id>/tree/
```

Named and anonymous volumes use `data.tar`. Bind mounts use `tree/`. Each
`meta.json` records the project identity and each volume's name, type, source,
stored bytes, checksum, and format.

## Safety and Limitations

- Restore is destructive and overwrites current volume contents without prompting.
- DockSnap refuses to restore a snapshot created for a different project name or working directory.
- Pausing containers provides a write freeze, not a crash-consistent database backup.
- DockSnap does not run `pg_dump`, `mysqldump`, or another database-aware backup tool.
- Bind mounts containing source code are included when they are attached to the project.
- Snapshots are full copies, not incremental snapshots.
- Restore starts existing containers instead of recreating containers or networks.
- Named-volume operations may pull `docker.io/library/alpine:3.21` as a helper image.

## How It Works

DockSnap talks directly to the Docker Engine API instead of invoking the Docker
CLI. Named volumes are mounted into a temporary Alpine helper container and
transferred through the Engine archive API. Bind mounts use `rsync` when
available and a Go filesystem copy otherwise. Snapshot metadata is committed by
renaming a temporary snapshot directory into place.

## Development

```bash
go test ./...
go build ./...
```

| Path | Responsibility |
| --- | --- |
| `cmd/dosnap` | Cobra commands and save/restore/list orchestration |
| `internal/engine` | Docker Engine client, interface, and fake |
| `internal/project` | Compose project and mount resolution |
| `internal/freeze` | Pause, unpause, stop, and start policies |
| `internal/copy` | Named-volume and bind-mount copy strategies |
| `internal/store` | Tags, metadata, snapshot storage, and listing |
| `internal/tui` | Interactive restore picker |
