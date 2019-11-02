package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
)

type downloadStats struct {
	NewFilesDownloaded   int
	VideosDownloaded     int
	MetadataDownloaded   int
	ThumbnailsDownloaded int
}

type chaninfo struct {
	Title string
	URL   string
}

func main() {
	app := cli.NewApp()
	app.Name = `youtube-chansync`
	app.Usage = `YTDL YouTube channel downloader`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `ytdl-bin, y`,
			Usage: `The command or path to the "youtube-dl" binary`,
			Value: `youtube-dl`,
		},
		cli.StringFlag{
			Name:  `channel-dir, c`,
			Usage: `The path containing existing downloaded channels.`,
			Value: `/cortex/videos/lib/youtube`,
		},
		cli.StringFlag{
			Name:  `nfofile`,
			Usage: `The name of the channel info file (.nfo) under the channel directory.`,
			Value: `youtube.nfo`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		chanGlob := `*`

		if c.NArg() > 0 {
			chanGlob = c.Args().First()
		}

		if ls, err := filepath.Glob(filepath.Join(
			c.String(`channel-dir`),
			chanGlob,
		)); err == nil {
			for _, dir := range ls {
				name := filepath.Base(dir)

				if stats, err := syncChannel(c, dir); err == nil {
					log.Debugf(
						"* channel: %s new=%d videos=%d meta=%d thumbs=%d",
						name,
						stats.NewFilesDownloaded,
						stats.VideosDownloaded,
						stats.MetadataDownloaded,
						stats.ThumbnailsDownloaded,
					)

					log.Infof("Channel %s synced successfully", name)
				} else {
					log.Error(err)
				}
			}
		} else {
			log.Fatal(err)
		}

		// for each actual dir in `channel-dir`
		// syncChannel(dir)

	}

	app.Run(os.Args)
}

func syncChannel(c *cli.Context, chanpath string) (downloadStats, error) {
	stats := downloadStats{}
	name := filepath.Base(chanpath)
	nfofile := filepath.Join(chanpath, c.String(`nfofile`))

	log.Infof("Syncing channel %s", name)
	log.Debugf("* chandir: %s", chanpath)

	if fileutil.IsNonemptyFile(nfofile) {
		if nfo, err := parseChannelInfo(name, nfofile); err == nil {
			log.Debugf("*   title: %s", nfo.Title)
			log.Debugf("*     url: %s", nfo.URL)

			return stats, nil
		} else {
			return stats, err
		}
	} else {
		return stats, fmt.Errorf("Cannot sync channel %q: missing infofile", name)
	}
}

func parseChannelInfo(name string, nfofile string) (*chaninfo, error) {
	if lines, err := fileutil.ReadAllLines(nfofile); err == nil {
		nfo := new(chaninfo)

		for _, line := range lines {
			key, value := stringutil.SplitPair(strings.TrimSpace(line), `=`)
			key = strings.ToLower(key)

			value = stringutil.Unwrap(value, `'`, `'`)
			value = stringutil.Unwrap(value, `"`, `"`)
			value = strings.TrimSpace(value)

			switch key {
			case `name`:
				nfo.Title = value
			case `url`:
				nfo.URL = value
			}
		}

		if nfo.Title == `` {
			nfo.Title = name
		}

		if nfo.URL == `` {
			return nil, fmt.Errorf("Unspecified channel URL, cannot sync")
		}

		return nfo, nil
	} else {
		return nil, fmt.Errorf("Cannot sync channel %q: cannot read infofile: %v", name, err)
	}
}
