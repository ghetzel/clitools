package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func main() {
	app := cli.NewApp()
	app.Name = `json`
	app.Usage = `Build and submit JSON documents`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `url, u`,
			Usage: `If specified, the generated JSON payload will be submitted to the given URL.`,
		},
		cli.StringFlag{
			Name:  `method, m`,
			Usage: `If submitting as an HTTP request, this is the HTTP method to use.`,
			Value: `post`,
		},
		cli.StringFlag{
			Name:  `content-type, t`,
			Usage: `If submitting as an HTTP request, this is the Content-Type to use.`,
			Value: `application/json; charset=utf-8`,
		},
		cli.StringSliceFlag{
			Name:  `param, p`,
			Usage: `If submitting as an HTTP request, include this key=value pair as a query string parameter.`,
		},
		cli.StringSliceFlag{
			Name:  `header, H`,
			Usage: `If submitting as an HTTP request, include this key=value pair as a request header value.`,
		},
		cli.StringFlag{
			Name:  `username, U`,
			Usage: `If submitting as an HTTP request, supply this username for Basic Authentication.`,
		},
		cli.StringFlag{
			Name:  `password, P`,
			Usage: `If submitting as an HTTP request, supply this password for Basic Authentication.`,
		},
		cli.IntFlag{
			Name:  `indent, I`,
			Usage: `The number of spaces to indent each successive level of the output object.`,
			Value: 4,
		},
		cli.BoolFlag{
			Name:  `insecure, k`,
			Usage: `Whether to ignore SSL server certificates that do not validate against the set of local trusted certificates.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		var out []byte
		var err error

		data := sliceOfPairsToMap(c.Args())
		indent := c.Int(`indent`)
		url := c.String(`url`)

		if url != `` {
			out, err = json.Marshal(data)
		} else {
			out, err = json.MarshalIndent(data, ``, strings.Repeat(` `, indent))
		}

		if err == nil {
			if url == `` {
				fmt.Println(string(out))
			} else if client, err := httputil.NewClient(url); err == nil {
				client.SetInsecureTLS(c.Bool(`insecure`))

				method := strings.ToUpper(c.String(`method`))
				params := sliceOfPairsToMap(c.StringSlice(`param`))
				headers := sliceOfPairsToMap(c.StringSlice(`header`))

				if c.IsSet(`username`) || c.IsSet(`password`) {
					client.SetBasicAuth(c.String(`username`), c.String(`password`))
				}

				if ct := c.String(`content-type`); ct != `` {
					headers[`Content-Type`] = ct
				}

				client.SetPreRequestHook(func(req *http.Request) (interface{}, error) {
					log.Infof("> HTTP %v %v", req.Method, req.URL)
					return req, nil
				})

				res, err := client.Request(httputil.Method(method), ``, data, params, headers)

				if res != nil {
					defer res.Body.Close()
					io.Copy(os.Stdout, res.Body)
					fmt.Println(``)
				}

				if err == nil {
					log.Infof("< %v", res.Status)
				} else {
					log.Fatalf("< %v", err)
				}
			} else {
				log.Fatalf("client error: %v", err)
			}
		} else {
			log.Fatalf("encode error: %v", err)
		}
	}

	app.Run(os.Args)
}

func sliceOfPairsToMap(pairs []string) map[string]interface{} {
	out := make(map[string]interface{})

	for _, pair := range pairs {
		k, v := stringutil.SplitPair(pair, `=`)
		maputil.DeepSet(out, strings.Split(k, `.`), stringutil.Autotype(v))
	}

	return out
}
