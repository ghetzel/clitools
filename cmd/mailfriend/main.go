package main

import (
	"fmt"
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
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
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  `uid, u`,
					Usage: `Only list UIDs of messages.`,
				},
			},
			Action: func(c *cli.Context) {
				if c.NArg() > 0 {
					if folder, err := profile.GetFolder(c.Args().First()); err == nil {
						for message := range folder.Messages() {
							clitools.Print(c, message, func() {
								if c.Bool(`uid`) {
									fmt.Println(message.ID())
								} else {
									fmt.Println(message.String())
								}
							})
						}
					} else {
						log.Fatalf("Cannot list folder: %v", err)
					}
				} else {
					if folders, err := profile.ListFolders(); err == nil {
						clitools.Print(c, folders, nil)
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
							clitools.Print(c, stat, nil)
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

						clitools.Print(c, metastat, nil)
					} else {
						log.Fatalf("Cannot get folders: %v", err)
					}
				}
			},
		}, {
			Name:      `rm`,
			ArgsUsage: `FOLDER`,
			Flags:     []cli.Flag{},
			Action: func(c *cli.Context) {
				if c.NArg() > 0 {
					if folder, err := profile.GetFolder(c.Args().First()); err == nil {
						msgchan := make(chan *Message)
						running := true

						go func() {
							for running {
								var bulk []*Message

								for msg := range msgchan {
									bulk = append(bulk, msg)

									if n := c.Int(`expunge-every`); len(bulk) >= n {
										log.Infof("[%v] Deleting %d messages", folder.Name, n)

										if err := folder.Delete(bulk...); err != nil {
											log.Warningf("Deletes failed: %v", err)
										}

										if err := folder.Expunge(); err != nil {
											log.Warningf("Expunge failed: %v", err)
										}

										bulk = nil
									}
								}
							}
						}()

						for message := range folder.Messages() {
							msgchan <- message
							log.Debugf("Dispatched message %v", message.Seq())
						}

						close(msgchan)

						log.Infof("Expunging folder %q", folder.Name)
						running = false
					} else {
						log.Fatalf("Cannot list folder: %v", err)
					}
				} else {
					log.Fatalf("Must specify a folder")
				}
			},
		},
	}

	app.Run(os.Args)
}
