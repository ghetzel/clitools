package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func main() {
	app := cli.NewApp()
	app.Name = `scut`
	app.Usage = `Like GNU cut, but supports regexp delimiters.`
	app.ArgsUsage = `[FILENAME]`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `delimiter, d`,
			Usage: `A delimiter string or /regular expression/ to split lines by.`,
			Value: `/\s+/`,
		},
		cli.StringFlag{
			Name:  `fields, f`,
			Usage: `Specify the fields, field ranges, and order that fields should be emitted in.`,
		},
		cli.BoolFlag{
			Name:  `only-delimited, s`,
			Usage: `Do not print lines not containing delimiters`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		ofs := os.Getenv(`OFS`)

		if ofs == `` {
			ofs = "\t"
		}

		delim := c.String(`delimiter`)
		fieldsExpr := c.String(`fields`)
		fields := make([]int, 0)

		for _, fexpr := range strings.Split(fieldsExpr, `,`) {
			if typeutil.IsInteger(fexpr) {
				fields = append(fields, int(typeutil.Int(fexpr)))
			} else {
				from, to := stringutil.SplitPair(fexpr, `-`)

				// this is a dirty hack. fight me.
				if from == `` {
					from = `0`
				}

				// this is a dirty hack. fight me.
				if to == `` {
					to = `999`
				}

				for i := int(typeutil.Int(from)); i <= int(typeutil.Int(to)); i++ {
					fields = append(fields, i+1)
				}
			}
		}

		var input io.ReadCloser

		if c.NArg() > 0 && c.Args().First() != `-` {
			if file, err := os.Open(c.Args().First()); err == nil {
				input = file
			} else {
				log.Fatal(err)
			}
		} else {
			input = os.Stdin
		}

		defer input.Close()

		lines := bufio.NewScanner(input)

		for lines.Scan() {
			line := lines.Text()
			var parts []string

			out := make([]string, 0)

			if stringutil.IsSurroundedBy(delim, `/`, `/`) {
				rx := regexp.MustCompile(stringutil.Unwrap(delim, `/`, `/`))

				if rx.MatchString(line) {
					parts = rx.Split(line, -1)
				} else if !c.Bool(`only-delimited`) {
					out = []string{line}
				} else {
					continue
				}
			} else {
				if strings.Contains(line, delim) {
					parts = strings.Split(line, delim)
				} else if !c.Bool(`only-delimited`) {
					out = []string{line}
				} else {
					continue
				}
			}

			if len(out) == 0 {
				for i, part := range parts {
					if len(fields) == 0 || sliceutil.Contains(fields, i+1) {
						out = append(out, part)
					}
				}
			}

			if len(out) > 0 {
				fmt.Println(strings.Join(out, ofs))
			}
		}

		if err := lines.Err(); err != nil {
			log.Fatal(err)
		}
	}

	app.Run(os.Args)
}
