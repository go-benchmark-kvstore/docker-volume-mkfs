package main

import (
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"syscall"

	"github.com/docker/go-plugins-helpers/volume"
	"gitlab.com/tozd/go/errors"
)

var _ volume.Driver = (*Driver)(nil)

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]$`)

// Capabilities implements volume.Driver.
func (*Driver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}

// Create implements volume.Driver.
func (d *Driver) Create(req *volume.CreateRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !nameRegexp.MatchString(req.Name) {
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

func (d *Driver) create(partition string) errors.E {
	// TODO: Redirect stdout and stderr to logger.
	cmd := exec.Command("mkfs.xfs", "-f", partition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return errors.WithStack(cmd.Run())
}

// Get implements volume.Driver.
func (d *Driver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !nameRegexp.MatchString(req.Name) {
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
func (d *Driver) List() (*volume.ListResponse, error) {
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
func (d *Driver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !nameRegexp.MatchString(req.Name) {
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
func (d *Driver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !nameRegexp.MatchString(req.Name) {
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
func (d *Driver) Remove(req *volume.RemoveRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !nameRegexp.MatchString(req.Name) {
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
func (d *Driver) Unmount(req *volume.UnmountRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !nameRegexp.MatchString(req.Name) {
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
	return errors.WithStack(syscall.Unmount(path.Join(d.Dir, name)))
}
