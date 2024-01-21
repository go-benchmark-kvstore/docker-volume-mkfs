# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Remove hard-coded `xfs` to really support `ext4`.

## [0.2.1] - 2024-01-21

### Fixed

- Add `e2fsprogs` to Docker (plugin) image.

## [0.2.0] - 2024-01-20

### Added

- Volume option `fs` to choose between `ext4` and `xfs` file systems. The default is
  `ext4` unless is changed using the CLI argument to the volume plugin executable.

### Changed

- Rename plugin CLI arguments from `partitions` to `args` because they in fact accept
  more arguments than just partitions.

## [0.1.0] - 2024-01-19

### Added

- First public release.

[unreleased]: https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/-/compare/v0.2.1...main
[0.2.1]: https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/-/compare/v0.2.0...v0.2.1
[0.2.0]: https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/-/compare/v0.1.0...v0.2.0
[0.1.0]: https://gitlab.com/go-benchmark-kvstore/docker-volume-mkfs/-/tags/v0.1.0

<!-- markdownlint-disable-file MD024 -->
