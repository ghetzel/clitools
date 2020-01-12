package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Wing924/shellwords"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var dryRun bool

type downloadStats struct {
	NewFilesDownloaded   int
	VideosDownloaded     int
	MetadataDownloaded   int
	SubtitlesDownloaded  int
	ThumbnailsDownloaded int
}

type chaninfo struct {
	Title string
	URL   string
	Args  []string
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
			Value:  `info`,
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
		cli.StringFlag{
			Name:  `archive-file, A`,
			Usage: `The name of the file containing a list of IDs already downloaded (inside each channel directory)`,
			Value: `archive.txt`,
		},
		cli.BoolFlag{
			Name:  `dry-run, n`,
			Usage: `Don't actually write anything to disk.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		dryRun = c.Bool(`dry-run`)

		return nil
	}

	app.Action = func(c *cli.Context) {
		chanGlob := `*`

		if c.NArg() > 0 {
			chanGlob = c.Args().First()
		}

		if ytdl := executil.Which(c.String(`ytdl-bin`)); ytdl != `` {
			if out, err := executil.Command(ytdl, `--version`).CombinedOutput(); err == nil {
				log.Infof("youtube-dl: %s (v%s)", ytdl, strings.TrimSpace(string(out)))
			} else {
				log.Fatalf("youtube-dl: failed to get version: %v", err)
			}

			if ls, err := filepath.Glob(filepath.Join(
				c.String(`channel-dir`),
				chanGlob,
			)); err == nil {
				for _, dir := range ls {
					name := filepath.Base(dir)

					stats, err := syncChannel(c, ytdl, dir)
					log.Debugf(
						"* channel: %s new=%d videos=%d meta=%d thumbs=%d subs=%d",
						name,
						stats.NewFilesDownloaded,
						stats.VideosDownloaded,
						stats.MetadataDownloaded,
						stats.ThumbnailsDownloaded,
						stats.SubtitlesDownloaded,
					)

					if stats.VideosDownloaded > 0 {
						log.Noticef("[channel] %s: added %d", name, stats.VideosDownloaded)
					}

					if err != nil {
						log.Error(err)
					}
				}
			} else {
				log.Fatal(err)
			}
		} else {
			log.Fatalf("youtube-dl: failed to locate youtube-dl binary")
		}
	}

	app.Commands = []cli.Command{
		{
			Name:      `rename`,
			Usage:     `Runs a rename pass on the given channel or all channels.`,
			ArgsUsage: `[CHANNEL ..]`,
			Action: func(c *cli.Context) {
				chanGlob := `*`

				if c.NArg() > 0 {
					chanGlob = c.Args().First()
				}

				if ls, err := filepath.Glob(filepath.Join(
					c.GlobalString(`channel-dir`),
					chanGlob,
				)); err == nil {
					for _, dir := range ls {
						if err := renameFilesIn(dir); err != nil {
							log.Fatalf("path %s: %v", dir, err)
						}
					}
				} else {
					log.Fatal(err)
				}
			},
		},
	}

	app.Run(os.Args)
}

func syncChannel(c *cli.Context, ytdl string, chanpath string) (downloadStats, error) {
	stats := downloadStats{}
	name := filepath.Base(chanpath)
	nfofile := filepath.Join(chanpath, c.String(`nfofile`))

	log.Infof("[channel] syncing %s", name)
	log.Debugf("* chandir: %s", chanpath)

	if fileutil.IsNonemptyFile(nfofile) {
		if nfo, err := parseChannelInfo(name, nfofile); err == nil {
			log.Debugf("*   title: %s", nfo.Title)
			log.Debugf("*     url: %s", nfo.URL)

			dlArgs := []string{
				`--verbose`,
				`--ignore-errors`,
				`--no-color`,
				`--no-call-home`,
				`--add-metadata`,
				`--output`, fmt.Sprintf("%s - %%(upload_date)s - %%(title)s.%%(ext)s", nfo.Title),
				`--format`, `best`,
				`--write-info-json`,
				`--write-auto-sub`,
				`--write-thumbnail`,
				`--sub-lang`, `en`,
				`--sub-format`, `best`,
				`--download-archive`, c.String(`archive-file`),
			}

			if dryRun {
				dlArgs = append(dlArgs, `--simulate`)
			}

			dlArgs = append(dlArgs, nfo.Args...)
			dlArgs = append(dlArgs, nfo.URL)

			// download new videos, thumbnails, and metadata
			dl := executil.Command(ytdl, dlArgs...)
			dl.Dir = chanpath
			outparse := func(line string, serr bool) {
				ll := log.DEBUG

				if serr || strings.HasPrefix(line, `[warn`) {
					ll = log.WARNING
				} else if strings.HasPrefix(line, `[err`) {
					ll = log.ERROR
				}

				log.Logf(ll, "    | %v", line)

				if strings.Contains(line, `Writing video subtitles to: `) {
					stats.SubtitlesDownloaded += 1
					stats.NewFilesDownloaded += 1
				} else if strings.Contains(line, `Writing video description metadata as JSON to: `) {
					stats.MetadataDownloaded += 1
					stats.NewFilesDownloaded += 1
				} else if strings.Contains(line, `Writing thumbnail to: `) {
					stats.ThumbnailsDownloaded += 1
					stats.NewFilesDownloaded += 1
				} else if strings.HasPrefix(line, `[download] Destination: `) {
					stats.VideosDownloaded += 1
					stats.NewFilesDownloaded += 1
				}
			}

			dl.OnStdout = outparse
			dl.OnStderr = outparse

			defer renameFilesIn(chanpath)

			if err := dl.Run(); err == nil {
				log.Debugf("youtube-dl completed successfully")
			} else {
				return stats, err
			}

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
			case `chan_args`:
				if words, err := shellwords.Split(value); err == nil {
					nfo.Args = words
				} else {
					log.Warningf("%s: invalid CHAN_ARGS: %v", filepath.Base(nfofile), err)
				}
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

func renameFilesIn(chanpath string) error {
	uniques := make(map[string]bool)

	if entries, err := filepath.Glob(filepath.Join(chanpath, `*.*`)); err == nil {
		for _, entry := range entries {
			entry = filepath.Base(entry)

			entry = strings.TrimSuffix(entry, `.info.json`)
			entry = strings.TrimSuffix(entry, `.en.vtt`)
			entry = strings.TrimSuffix(entry, `-thumb.jpg`)
			entry = strings.TrimSuffix(entry, filepath.Ext(entry))

			uniques[entry] = true
		}

		bases := maputil.StringKeys(uniques)
		sort.Strings(bases)

		for _, base := range bases {
			infoJson := filepath.Join(chanpath, fmt.Sprintf("%s.info.json", base))

			if info, err := os.Open(infoJson); err == nil {
				var meta ytdlInfo
				defer info.Close()

				if err := json.NewDecoder(info).Decode(&meta); err == nil {
					if yr := meta.Field(`year`); yr != nil {
						title := typeutil.String(meta.Field(`title`))

						if len(title) < 3 {
							title = `(unknown title)`
						}

						mn := meta.Field(`month`)
						dy := meta.Field(`day`)
						id := meta.Field(`id`)
						newbase := fmt.Sprintf("%s-%s-%s - %s (%s)", yr, mn, dy, title, id)

						if err := renameFilesForItem(chanpath, base, newbase); err != nil {
							return err
						}
					} else {
						log.Warningf("%s: invalid upload_date", base)
					}
				} else {
					return err
				}

				info.Close()
			} else if os.IsNotExist(err) {
				continue
			} else {
				return err
			}
		}

		return nil
	} else {
		return err
	}
}

func getVideoExtension(parent string, base string) string {
	if entries, err := filepath.Glob(filepath.Join(parent, base+`.*`)); err == nil {
		for _, entry := range entries {
			mt, _ := stringutil.SplitPair(fileutil.GetMimeType(entry), `/`)

			if mt == `video` {
				return filepath.Ext(entry)
			}
		}
	}

	return ``
}

func renameFilesForItem(parent string, base string, newbase string) error {
	if entries, err := filepath.Glob(filepath.Join(parent, base+`.*`)); err == nil {
		for _, entry := range entries {
			dir, file := filepath.Split(entry)
			ext := strings.TrimPrefix(file, base)
			to := filepath.Join(dir, newbase+ext)

			if entry != to {
				if !dryRun {
					if err := os.Rename(entry, to); err != nil {
						return err
					}
				} else {
					log.Noticef("dry-run: would mv %s/%s", filepath.Base(parent), file)
					log.Noticef("               to %s/%s", filepath.Base(parent), newbase+ext)
				}
			}
		}
	}

	return nil
}
