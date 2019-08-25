package main

import (
	"bytes"
	"fmt"
	"os"
	"syscall"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	client := NewClient()

	app := cli.NewApp()
	app.Name = `ofxtool`
	app.Usage = `Utility for interacting with and managing Open Financial eXchange (OFX) datasources`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `format, f`,
			Usage: `Specify the output format for data output from subcommands.`,
			Value: `json`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		log.FatalIf(client.Connect())

		return nil
	}

	// app.Action = func(c *cli.Context) {}

	app.Commands = []cli.Command{
		{
			Name:  `list`,
			Usage: `List all saved institutions`,
			Action: func(c *cli.Context) {
				if institutions, err := client.Institutions(); err == nil {
					clitools.Print(c, institutions, nil)
				} else {
					log.Fatal(err)
				}
			},
		}, {
			Name:      `create`,
			Usage:     `Register a new institution.`,
			ArgsUsage: `USERNAME`,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  `ofxhome-id, i`,
					Usage: `Specify the ofxhome.com Institution ID to populate the OFX data from.`,
				},
				cli.StringFlag{
					Name:  `name, n`,
					Usage: `Set or override the user-facing name.`,
				},
				cli.StringFlag{
					Name:  `url, u`,
					Usage: `Set or override the OFX URL`,
				},
				cli.StringFlag{
					Name:  `organization, o`,
					Usage: `Set or override the OFX ORGID`,
				},
				cli.IntFlag{
					Name:  `fid, f`,
					Usage: `Set or override the OFX FID`,
				},
			},
			Action: func(c *cli.Context) {
				if c.NArg() < 1 {
					log.Fatalf("Must provide LABEL and USERNAME as positional arguments")
				}

				institution := Institution{
					Username: c.Args().First(),
				}

				if ohid := c.Int(`ofxhome-id`); ohid > 0 {
					institution.OHID = ohid

					if err := PopulateFromOfxHome(&institution, ohid); err != nil {
						log.Fatalf("ofxhome: %v", err)
					}
				}

				if v := c.String(`name`); v != `` {
					institution.Name = v
				}

				if v := c.String(`url`); v != `` {
					institution.URL = v
				}

				if v := c.String(`organization`); v != `` {
					institution.Organization = v
				}

				if v := c.Int(`fid`); v > 0 {
					institution.FID = v
				}

				fmt.Print(" Enter Password: ")

				if pass1, err := terminal.ReadPassword(int(syscall.Stdin)); err == nil {
					fmt.Print("\nVerify Password: ")

					if pass2, err := terminal.ReadPassword(int(syscall.Stdin)); err == nil {
						if bytes.Equal(pass1, pass2) {
							if err := client.CreateInstitution(&institution, string(pass1)); err == nil {
								log.Noticef("Institution %v created successfully", institution)
							} else {
								log.Fatalf("create: %v", err)
							}
						} else {
							log.Fatal("Passwords do not match")
						}
					} else {
						log.Fatal("pass: %v", err)
					}
				} else {
					log.Fatal("pass: %v", err)
				}
			},
		},
	}

	app.Run(os.Args)
}
