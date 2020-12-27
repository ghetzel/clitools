package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
			Value:  `warning`,
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

		var data = sliceOfPairsToMap(c.Args())
		var indent = c.Int(`indent`)
		var baseurl = c.String(`url`)

		if baseurl != `` {
			out, err = json.Marshal(data)
		} else {
			out, err = json.MarshalIndent(data, ``, strings.Repeat(` `, indent))
		}

		if err == nil {
			if baseurl == `` {
				fmt.Println(string(out))
				return
			}

			var addHeaders = make(map[string]interface{})

			if u, err := url.Parse(baseurl); err == nil {
				switch u.Scheme {
				case `kubernetes`:
					baseurl = os.ExpandEnv("https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT_HTTPS}/")
					addHeaders[`Authorization`] = os.ExpandEnv("Bearer ${KUBERNETES_API_TOKEN}")
				}
			}

			if client, err := httputil.NewClient(baseurl); err == nil {
				client.SetInsecureTLS(c.Bool(`insecure`))
				client.SetAutomaticLogin(true)

				for k, v := range addHeaders {
					client.SetHeader(k, v)
				}

				client.SetPreRequestHook(func(req *http.Request) (interface{}, error) {
					log.Debugf("[http] > %v %v", req.Method, req.URL)

					for k, vs := range req.Header {
						switch k {
						case `Authorization`:
							continue
						}

						for _, v := range vs {
							log.Debugf("[http] >   %v: %v", k, v)
						}
					}

					return nil, nil
				})

				client.SetPostRequestHook(func(res *http.Response, _ interface{}) error {
					log.Debugf("[http] < %s %s", res.Proto, res.Status)

					for k, vs := range res.Header {
						for _, v := range vs {
							log.Debugf("[http] <  %v: %v", k, v)
						}
					}

					return nil
				})

				method := strings.ToUpper(c.String(`method`))
				params := sliceOfPairsToMap(c.StringSlice(`param`))
				headers := sliceOfPairsToMap(c.StringSlice(`header`))

				if c.IsSet(`username`) || c.IsSet(`password`) {
					client.SetBasicAuth(c.String(`username`), c.String(`password`))
				}

				if ct := c.String(`content-type`); ct != `` {
					headers[`Content-Type`] = ct
				}

				res, err := client.Request(httputil.Method(method), ``, data, params, headers)

				if res != nil {
					defer res.Body.Close()
					io.Copy(os.Stdout, res.Body)
					fmt.Println(``)
				}

				if err != nil {
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
		k, typ := stringutil.SplitPair(k, `:`)
		var vT interface{}
		var converted bool

		if tt := stringutil.ParseType(typ); tt != stringutil.Invalid {
			if vv, err := stringutil.ConvertTo(tt, v); err == nil {
				vT = vv
				converted = true
			}
		}

		if !converted {
			vT = stringutil.Autotype(v)
		}

		maputil.DeepSet(out, strings.Split(k, `.`), vT)
	}

	return out
}
