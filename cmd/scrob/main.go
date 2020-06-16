package main

import (
	"os"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/jdxcode/netrc"
	"github.com/shkh/lastfm-go/lastfm"
)

func main() {
	app := cli.NewApp()
	app.Name = `scrob`
	app.Usage = `A command line last.fm scrobbler.`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `debug`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `title, t`,
			Usage: `Track title`,
		},
		cli.StringFlag{
			Name:  `album, a`,
			Usage: `Album name`,
		},
		cli.StringFlag{
			Name:  `artist, A`,
			Usage: `Artist name`,
		},
		cli.IntFlag{
			Name:  `year, Y`,
			Usage: `Release year`,
		},
		cli.IntFlag{
			Name:  `track, T`,
			Usage: `Track number`,
		},
		cli.IntFlag{
			Name:  `disc, D`,
			Usage: `Disc number`,
			Value: 1,
		},
		cli.IntFlag{
			Name:  `timestamp, x`,
			Usage: `Unix timestamp representing when playback started.`,
			Value: int(time.Now().Unix()),
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		log.Fatal("NOT YET IMPLEMENTED")

		var token string = os.Getenv(`SCROB_LASTFM_USER`)
		var secret string = os.Getenv(`SCROB_LASTFM_PASS`)

		if credfile := fileutil.MustExpandUser(`~/.netrc`); fileutil.FileExists(credfile) {
			if nrc, err := netrc.Parse(credfile); err == nil {
				if machine := nrc.Machine(`last.fm`); machine != nil {
					token = machine.Get(`login`)
					secret = machine.Get(`password`)
				}
			} else {
				log.Fatal(err)
			}
		}

		var api = lastfm.New(token, secret)
		log.FatalIf(api.Login(token, secret))

		var metadata = map[string]interface{}{
			`title`:     c.String(`title`),
			`artist`:    c.String(`artist`),
			`album`:     c.String(`album`),
			`track`:     c.Int(`track`),
			`disc`:      c.Int(`disc`),
			`timestamp`: c.String(`timestamp`),
		}

		if scrobble, err := api.Track.Scrobble(metadata); err == nil {
			log.Infof("scrobble: acc=%s ign=%s", scrobble.Accepted, scrobble.Ignored)
		} else {
			log.Fatal(err)
		}
	}

	app.Run(os.Args)
}
