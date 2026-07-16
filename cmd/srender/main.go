package main

import (
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
)

func main() {
	app := cli.NewApp()
	app.Name = `srender`
	app.Usage = `Template renderer utility`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {

	}

	app.Run(os.Args)
}
