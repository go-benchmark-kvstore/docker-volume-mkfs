package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"syscall"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/moby/moby/daemon/names"
	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

var _ volume.Driver = (*Driver)(nil)

// Capabilities implements volume.Driver.
func (d *Driver) Capabilities() *volume.CapabilitiesResponse {
	d.Logger.Debug().Msg("capabilities")

	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}

// Create implements volume.Driver.
func (d *Driver) Create(req *volume.CreateRequest) (err error) {
	d.Logger.Debug().Str("name", req.Name).Interface("options", req.Options).Msg("create")

	defer func() {
		if err != nil {
			d.Logger.Error().Str("name", req.Name).Interface("options", req.Options).Err(err).Msg("create")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	if !names.RestrictedNamePattern.MatchString(req.Name) {
		return errors.New("invalid volume name")
	}
	if _, ok := d.volumes[req.Name]; ok {
		return errors.New("volume already exists")
	}

	availablePartitions := []string{}
	for _, partition := range d.Partitions {
		found := false
		for _, p := range d.volumes {
			if partition == p {
				found = true
				break
			}
		}
		if !found {
			availablePartitions = append(availablePartitions, partition)
		}
	}

	if len(availablePartitions) == 0 {
		return errors.New("no unused partitions left")
	}

	errE := d.create(availablePartitions[0])
	if errE != nil {
		return errE
	}

	d.volumes[req.Name] = availablePartitions[0]
	d.mounts[req.Name] = []string{}

	return nil
}

func (d *Driver) redirectToLogger(command, partition, name string, level zerolog.Level, reader io.Reader) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			d.Logger.WithLevel(level).Str("command", command).Str("partition", partition).Msg(line)
		}
	}

	err := scanner.Err()
	// Reader can get closed and we ignore that.
	if err != nil && !errors.Is(err, os.ErrClosed) {
		d.Logger.Warn().Str("command", command).Str("partition", partition).Err(err).Msgf("error reading %s", name)
	}
}

func (d *Driver) create(partition string) errors.E {
	cmd := exec.Command("mkfs.xfs", "-f", partition)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.WithStack(err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.WithStack(err)
	}
	defer stderr.Close()

	g := errgroup.Group{}

	g.Go(func() error {
		d.redirectToLogger("mkfs.xfs", partition, "stdout", zerolog.DebugLevel, stdout)
		return nil
	})
	g.Go(func() error {
		d.redirectToLogger("mkfs.xfs", partition, "stderr", zerolog.ErrorLevel, stderr)
		return nil
	})

	return errors.Join(g.Wait(), cmd.Wait())
}

// Get implements volume.Driver.
func (d *Driver) Get(req *volume.GetRequest) (_ *volume.GetResponse, err error) {
	d.Logger.Debug().Str("name", req.Name).Msg("get")

	defer func() {
		if err != nil {
			d.Logger.Error().Str("name", req.Name).Err(err).Msg("get")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	if !names.RestrictedNamePattern.MatchString(req.Name) {
		return nil, errors.New("invalid volume name")
	}
	if _, ok := d.volumes[req.Name]; !ok {
		return nil, errors.New("volume does not exist")
	}

	return &volume.GetResponse{
		Volume: &volume.Volume{ //nolint:exhaustruct
			Name:       req.Name,
			Mountpoint: path.Join(d.Dir, req.Name),
		},
	}, nil
}

// List implements volume.Driver.
func (d *Driver) List() (_ *volume.ListResponse, err error) {
	d.Logger.Debug().Msg("list")

	defer func() {
		if err != nil {
			d.Logger.Error().Err(err).Msg("list")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	volumes := []*volume.Volume{}
	for v := range d.volumes {
		volumes = append(volumes, &volume.Volume{ //nolint:exhaustruct
			Name:       v,
			Mountpoint: path.Join(d.Dir, v),
		})
	}

	return &volume.ListResponse{
		Volumes: volumes,
	}, nil
}

// Mount implements volume.Driver.
func (d *Driver) Mount(req *volume.MountRequest) (_ *volume.MountResponse, err error) {
	d.Logger.Debug().Str("name", req.Name).Str("id", req.ID).Msg("mount")

	defer func() {
		if err != nil {
			d.Logger.Error().Str("name", req.Name).Str("id", req.ID).Err(err).Msg("mount")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	if !names.RestrictedNamePattern.MatchString(req.Name) {
		return nil, errors.New("invalid volume name")
	}
	partition, ok := d.volumes[req.Name]
	if !ok {
		return nil, errors.New("volume does not exist")
	}

	if len(d.mounts[req.Name]) == 0 {
		errE := d.mount(partition, req.Name)
		if errE != nil {
			return nil, errE
		}
	}

	d.mounts[req.Name] = append(d.mounts[req.Name], req.ID)

	return &volume.MountResponse{
		Mountpoint: path.Join(d.Dir, req.Name),
	}, nil
}

func (d *Driver) mount(partition, name string) errors.E {
	p := path.Join(d.Dir, name)
	err := os.MkdirAll(p, 0o700) //nolint:gomnd
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(syscall.Mount(partition, p, "xfs", 0, ""))
}

// Path implements volume.Driver.
func (d *Driver) Path(req *volume.PathRequest) (_ *volume.PathResponse, err error) {
	d.Logger.Debug().Str("name", req.Name).Msg("path")

	defer func() {
		if err != nil {
			d.Logger.Error().Str("name", req.Name).Err(err).Msg("path")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	if !names.RestrictedNamePattern.MatchString(req.Name) {
		return nil, errors.New("invalid volume name")
	}
	if _, ok := d.volumes[req.Name]; !ok {
		return nil, errors.New("volume does not exist")
	}

	return &volume.PathResponse{
		Mountpoint: path.Join(d.Dir, req.Name),
	}, nil
}

// Remove implements volume.Driver.
func (d *Driver) Remove(req *volume.RemoveRequest) (err error) {
	d.Logger.Debug().Str("name", req.Name).Msg("remove")

	defer func() {
		if err != nil {
			d.Logger.Error().Str("name", req.Name).Err(err).Msg("remove")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	if !names.RestrictedNamePattern.MatchString(req.Name) {
		return errors.New("invalid volume name")
	}
	if _, ok := d.volumes[req.Name]; !ok {
		return errors.New("volume does not exist")
	}

	delete(d.volumes, req.Name)
	delete(d.mounts, req.Name)

	return nil
}

// Unmount implements volume.Driver.
func (d *Driver) Unmount(req *volume.UnmountRequest) (err error) {
	d.Logger.Debug().Str("name", req.Name).Str("id", req.ID).Msg("unmount")

	defer func() {
		if err != nil {
			d.Logger.Error().Str("name", req.Name).Str("id", req.ID).Err(err).Msg("unmount")
		}
	}()

	d.mu.Lock()
	defer d.mu.Unlock()

	if !names.RestrictedNamePattern.MatchString(req.Name) {
		return errors.New("invalid volume name")
	}
	if _, ok := d.volumes[req.Name]; !ok {
		return errors.New("volume does not exist")
	}

	i := slices.Index(d.mounts[req.Name], req.ID)
	if i == -1 {
		return errors.New("mount does not exist")
	}

	d.mounts[req.Name] = slices.Delete(d.mounts[req.Name], i, i+1)

	if len(d.mounts[req.Name]) == 0 {
		errE := d.umount(req.Name)
		if errE != nil {
			return errE
		}
	}

	return nil
}

func (d *Driver) umount(name string) errors.E {
	return errors.WithStack(syscall.Unmount(path.Join(d.Dir, name), 0))
}
