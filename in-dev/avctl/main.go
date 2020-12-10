package main

import (
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
)

func main() {
	app := cli.NewApp()
	app.Name = `avctl`
	app.Usage = `Command-line control for system volume and media players.`
	app.ArgsUsage = `[FILENAME]`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `debug`,
			EnvVar: `LOGLEVEL`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  `volume`,
			Usage: `Control the volume of the system audio or applications.`,
			Subcommands: []cli.Command{
				{
					Name:      `up`,
					Usage:     `Raise the volume`,
					ArgsUsage: `[PERCENT]`,
					Action: func(c *cli.Context) {
						log.Debugf("volume up")
					},
				}, {
					Name:      `down`,
					Usage:     `Lower the volume`,
					ArgsUsage: `[PERCENT]`,
					Action: func(c *cli.Context) {
						log.Debugf("volume down")
					},
				}, {
					Name:  `toggle`,
					Usage: `Mute or unmute the audio of the given output`,
					Action: func(c *cli.Context) {
						log.Debugf("volume toggle")
					},
				}, {
					Name:  `mute`,
					Usage: `Mute the given output`,
					Action: func(c *cli.Context) {
						log.Debugf("volume mute")
					},
				}, {
					Name:  `unmute`,
					Usage: `Unmute the given output`,
					Action: func(c *cli.Context) {
						log.Debugf("volume unmute")
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
