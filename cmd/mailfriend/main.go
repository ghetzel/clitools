package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/NonerKao/color-aware-tabwriter"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
)

func main() {
	app := cli.NewApp()
	app.Name = `mailfriend`
	app.Usage = `Utility for interacting with remote mailboxes.`
	app.Version = clitools.Version

	var profile *Profile

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `profile, p`,
			Usage: `Specifies the default profile to use for unqualified paths`,
			Value: DefaultProfileName,
		},
		cli.StringFlag{
			Name:  `format, f`,
			Usage: `Specify the output format for data output from subcommands.`,
			Value: `text`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))

		if p, err := NewProfile(c.String(`profile`)); err == nil {
			profile = p
		} else {
			log.Fatal(err)
		}

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:      `ls`,
			ArgsUsage: `[FOLDER]`,
			Flags:     []cli.Flag{},
			Action: func(c *cli.Context) {
				if c.NArg() > 0 {
					if folder, err := profile.GetFolder(c.Args().First()); err == nil {
						for message := range folder.Messages() {
							print(c, message, func() {
								fmt.Println(message.String())
							})
						}
					} else {
						log.Fatalf("Cannot list folder: %v", err)
					}
				} else {
					if folders, err := profile.ListFolders(); err == nil {
						print(c, folders, nil)
					} else {
						log.Fatalf("Cannot list folders: %v", err)
					}
				}
			},
		}, {
			Name:      `stat`,
			ArgsUsage: `[FOLDER]`,
			Flags:     []cli.Flag{},
			Action: func(c *cli.Context) {
				if c.NArg() > 0 {
					if folder, err := profile.GetFolder(c.Args().First()); err == nil {
						if stat, err := folder.Statistics(); err == nil {
							print(c, stat, nil)
						} else {
							log.Fatalf("Cannot stat folder: %v", err)
						}
					} else {
						log.Fatalf("Cannot get folder: %v", err)
					}
				} else {
					if folders, err := profile.ListFolders(); err == nil {
						metastat := new(FolderStats)

						for _, folder := range folders {
							if stat, err := folder.Statistics(); err == nil {
								metastat.Add(stat)
							} else {
								log.Warningf("Cannot stat folder %q: %v", folder.Name, err)
							}
						}

						print(c, metastat, nil)
					} else {
						log.Fatalf("Cannot get folders: %v", err)
					}
				}
			},
		},
	}

	app.Run(os.Args)
}

func print(c *cli.Context, data interface{}, txtfn func()) {
	if data != nil {
		switch c.GlobalString(`format`) {
		case `json`:
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent(``, `  `)
			enc.Encode(data)
		default:
			if txtfn != nil {
				txtfn()
			} else {
				tw := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

				for _, line := range sliceutil.Compact([]interface{}{data}) {
					fmt.Fprintf(tw, "%v\n", line)
				}

				tw.Flush()
			}
		}
	}
}
