package main

import (
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"

	"github.com/containerd/containerd"
)

func main() {
	var app = cli.NewApp()
	app.Name = `containerboi`
	app.Usage = `A containerd teaching app`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `debug`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `socket, S`,
			Usage: `Path to the containerd control socket.`,
			Value: fileutil.MustExpandUser(`~/containerboi.sock`),
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))

		return nil
	}

	app.Action = func(c *cli.Context) {
		var client, err = containerd.New(c.String(`socket`))
		log.FatalIf(err)
		defer client.Close()
	}

	app.Run(os.Args)
}
