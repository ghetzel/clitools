package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func main() {
	app := cli.NewApp()
	app.Name = `rxmv`
	app.Usage = `Bulk rename files using regular expressions.`
	app.ArgsUsage = `"s/FIND/REPLACE/" FILE [..]`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.BoolFlag{
			Name:  `yes, y`,
			Usage: `Assume "yes" to any questions.`,
		},
		cli.BoolFlag{
			Name:  `dry-run, n`,
			Usage: `Don't actually rename files, just show what would happen.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		if c.NArg() < 2 {
			cli.ShowAppHelp(c)
			return
		}

		spattern := strings.TrimPrefix(c.Args().First(), `s`)
		filenames := c.Args()[1:]

		for i, filename := range filenames {
			if fileutil.DirExists(filename) {
				if entries, err := ioutil.ReadDir(filename); err == nil {
					var pre []string
					var post []string
					var expanded []string

					pre = filenames[:i]

					if i+1 < len(filenames) {
						post = filenames[i+1:]
					}

					for _, entry := range entries {
						expanded = append(expanded, filepath.Join(filename, entry.Name()))
					}

					filenames = append(pre, expanded...)
					filenames = append(filenames, post...)
				} else {
					log.Fatalf("Cannot read directory %q: %v", filename, err)
				}
			}
		}

		if len(spattern) == 0 {
			log.Fatalf("malformed s/// pattern")
		} else {
			sep := string(spattern[0])
			find, repl, _ := stringutil.SplitTriple(spattern[1:], sep)

			if rx, err := regexp.Compile(find); err == nil {
				table := tabwriter.NewWriter(os.Stdout, 1, 4, 1, ' ', 0)
				moves := make(map[string]string)

				for _, before := range filenames {
					if rx.MatchString(before) {
						after := rx.ReplaceAllString(before, repl)
						fmt.Fprintf(table, "%s\t->\t%s\n", before, after)
						moves[before] = after
					}
				}

				if len(moves) == 0 {
					return
				}

				fmt.Fprintf(table, "\n")

				if !c.Bool(`yes`) {
					table.Flush()
				}

				if c.Bool(`dry-run`) {
					log.Noticef("Not doing anything.")
					return
				} else if c.Bool(`yes`) || log.Confirm("Proceed with renaming the above files? (y/n): ") {
					var wg sync.WaitGroup

					for b, a := range moves {
						wg.Add(1)

						go func(w *sync.WaitGroup, before string, after string) {
							log.Infof("Renaming %s -> %s", before, after)

							if err := os.Rename(before, after); err != nil {
								log.Errorf("%s: %v", before, err)
							}

							wg.Done()
						}(&wg, b, a)
					}

					wg.Wait()
				}
			} else {
				log.Fatalf("malformed pattern: %v", err)
			}
		}
	}

	app.Run(os.Args)
}
