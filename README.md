# docker-volume-mkfs

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/go-benchmark-kvstore/docker-volume-mkfs)](https://goreportcard.com/report/gitlab.com/go-benchmark-kvstore/docker-volume-mkfs)
[![pipeline status](https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/badges/main/pipeline.svg?ignore_skipped=true)](https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/-/pipelines)

[Docker volume plugin](https://docs.docker.com/engine/extend/legacy_plugins/) manages volumes by formatting partitions.
You provide it with a list of available partitions and to create a volume it formats the partition and Docker then mounts
it into your container as a volume. When volume is removed, partition is returned to available partitions, to be
reformatted for the next volume.

This allows one to always get a clean unfragmented volume and deleting the volume is quick. If
you have relatively short lived containers with millions of files created in a volume during their life time,
fragmentation and file deletion can become a problem.

The plugin was created to [make benchmarks better](https://gitlab.com/go-benchmark-kvstore/go-benchmark-kvstore).
Each benchmark run gets a fresh file system and cleanup after a run is quick.

Features:

- Creates volumes with freshly formatted partitions.
- Supports `ext4` (default, unless changed through CLI) and `xfs` file systems, which you can choose using
  volume option `fs`.
- Discarding volumes is quick (just unmount).
- When partition gets full, the volume gets full.

## Installation

[Releases page](https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/-/releases)
contains a list of stable versions. Each includes:

- Statically compiled binaries.
- Docker images.
- Docker plugin images.

You should just download/use the latest one.

The tool is implemented in Go. You can use Docker to install the plugin. For example, for `v0.1.0` version:

```sh
docker plugin install --alias mkfs --grant-all-permissions \
 registry.gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/plugin-tag/v0-1-0:latest \
 partitions="/dev/nvme0n1p1 /dev/nvme1n1p1 /dev/nvme2n1p1 /dev/nvme3n1p1"
```

To install the latest development version (`main` branch), use `registry.gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/plugin-branch/main:latest`
for its Docker plugin image. That allows you to upgrade the plugin when the `main` branch is updated:

```sh
docker plugin disable mkfs
docker plugin upgrade --grant-all-permissions mkfs
docker plugin enable mkfs
```

## Usage

You can use it for volumes of your container using `--volume-driver` argument:

```sh
docker run --volume-driver mkfs ...
```

You can also create a named volume:

```sh
docker volume create --driver mkfs --opt fs=xfs <name>
```

## GitHub mirror

There is also a [read-only GitHub mirror available](https://github.com/go-benchmark-kvstore/docker-volume-mkfs),
if you need to fork the project there.
