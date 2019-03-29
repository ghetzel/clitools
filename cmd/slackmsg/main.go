package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/jdxcode/netrc"
)

func main() {
	app := cli.NewApp()
	app.Name = `slackmsg`
	app.Usage = `Post a message to a Slack channel.`
	app.ArgsUsage = `MESSAGE`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `netrc`,
			Usage: `The path to the .netrc file to read.`,
			Value: `~/.netrc`,
		},
		cli.StringFlag{
			Name:  `netrc-machine`,
			Usage: `The "machine" to use when reading the netrc file.`,
			Value: `slack`,
		},
		cli.StringFlag{
			Name:  `url, u`,
			Usage: `The Slack App Incoming Webhook URL to use (overrides netrc values)`,
		},
		cli.StringFlag{
			Name:  `channel, c`,
			Usage: `The channel to post the message to.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		message := strings.Join(c.Args(), ` `)

		var hookPath string

		if message == `` {
			if fileutil.IsTerminal() {
				if m, err := ioutil.ReadAll(os.Stdin); err == nil {
					message = string(m)
				} else {
					log.Fatalf("read error: %v", err)
				}
			}
		}

		if message == `` {
			log.Fatalf("no message provided")
		}

		if hookUrl := c.String(`url`); hookUrl != `` {
			if strings.HasPrefix(hookUrl, `/`) {
				hookPath = hookUrl
			} else if u, err := url.Parse(hookUrl); err == nil {
				hookPath = u.Path
			} else {
				log.Fatalf("url: %v", err)
			}
		} else {
			if rc, err := netrc.Parse(fileutil.MustExpandUser(c.String(`netrc`))); err == nil {
				if machine := rc.Machine(c.String(`netrc-machine`)); machine != nil {
					if path := machine.Get(`password`); path != `` {
						hookPath = path
					}
				} else {
					log.Fatalf("netrc: no such machine %q", c.String(`netrc-machine`))
				}
			} else {
				log.Fatalf("client error: %v", err)
			}
		}

		if hookPath != `` {
			if client, err := httputil.NewClient(`https://hooks.slack.com`); err == nil {
				payload := map[string]interface{}{
					`text`: message,
				}

				if channel := c.String(`channel`); channel != `` {
					payload[`channel`] = fmt.Sprintf("#%s", strings.TrimPrefix(channel, `#`))
				}

				if username := c.String(`username`); username != `` {
					payload[`username`] = username
				}

				if emoji := c.String(`emoji`); emoji != `` {
					payload[`icon_emoji`] = emoji
				}

				if _, err := client.Post(hookPath, payload, nil, nil); err != nil {
					log.Fatalf("send failed: %v", err)
				}
			} else {
				log.Fatalf("client error: %v", err)
			}
		} else {

		}
	}

	app.Run(os.Args)
}
