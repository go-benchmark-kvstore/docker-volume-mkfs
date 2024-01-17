package main

import (
	"os/user"
	"strconv"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/docker/go-plugins-helpers/volume"
	"gitlab.com/tozd/go/cli"
	"gitlab.com/tozd/go/errors"
	"gitlab.com/tozd/go/zerolog"
)

//nolint:lll
type Driver struct {
	zerolog.LoggingConfig

	Partitions []string `arg:""                help:"Partition(s) to use."                                            name:"partition"                   required:""           type:"path"`
	Dir        string   `       default:"/mnt" help:"Directory under which to mount partitions. Default: ${default}."                  placeholder:"DIR"             short:"d" type:"path"`

	// Map between volume names and partitions.
	volumes map[string]string

	// Map between volume names and active mount IDs.
	mounts map[string][]string

	// Lock.
	mu sync.Mutex
}

func main() {
	var driver Driver
	cli.Run(&driver, nil, func(ctx *kong.Context) errors.E {
		driver.volumes = make(map[string]string)
		driver.mounts = make(map[string][]string)
		handler := volume.NewHandler(&driver)
		u, _ := user.Lookup("root")
		gid, _ := strconv.Atoi(u.Gid)
		driver.Logger.Debug().Str("path", "/run/docker/plugins/mkfs.sock").Msg("running")
		return errors.WithStack(handler.ServeUnix("mkfs", gid))
	})
}
