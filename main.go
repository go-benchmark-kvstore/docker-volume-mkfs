package main

import (
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/docker/go-plugins-helpers/volume"
	"gitlab.com/tozd/go/cli"
	"gitlab.com/tozd/go/errors"
	"gitlab.com/tozd/go/zerolog"
)

var fileSystems = map[string][]string{ //nolint:gochecknoglobals
	"ext4": {"mkfs.ext4", "-F"},
	"xfs":  {"mkfs.xfs", "-f"},
}

//nolint:lll
type Driver struct {
	zerolog.LoggingConfig

	Partitions []string `arg:""                                      help:"Partition(s) to use."                         name:"partition"                   required:""           type:"path"`
	Dir        string   `       default:"/mnt"                       help:"Directory under which to mount partitions."                    placeholder:"DIR"             short:"d" type:"path"`
	Default    string   `       default:"ext4" enum:"${fileSystems}" help:"Default file system to format partitions as."                  placeholder:"FS"`

	// Map between volume names and activeVolume.
	volumes map[string]activeVolume

	// Map between volume names and active mount IDs.
	mounts map[string][]string

	// Lock.
	mu sync.Mutex
}

func main() {
	names := []string{}
	for name := range fileSystems {
		names = append(names, name)
	}
	sort.Strings(names)
	var driver Driver
	cli.Run(&driver, kong.Vars{
		"fileSystems": strings.Join(names, ","),
	}, func(_ *kong.Context) errors.E {
		driver.volumes = make(map[string]activeVolume)
		driver.mounts = make(map[string][]string)
		handler := volume.NewHandler(&driver)
		u, _ := user.Lookup("root")
		gid, _ := strconv.Atoi(u.Gid)
		driver.Logger.Debug().Str("path", "/run/docker/plugins/mkfs.sock").Msg("running")
		return errors.WithStack(handler.ServeUnix("mkfs", gid))
	})
}
