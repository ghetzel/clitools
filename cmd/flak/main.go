package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

type sshResults struct {
	Status      int       `json:"status"`
	Stdout      []string  `json:"stdout"`
	Stderr      []string  `json:"stderr"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

func (self *sshResults) Duration() time.Duration {
	return self.CompletedAt.Sub(self.StartedAt)
}

func main() {
	app := cli.NewApp()
	app.Name = `flak`
	app.Usage = `A big stupid for-loop for running SSH commands`
	app.ArgsUsage = `COMMAND`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  `hosts, H`,
			Usage: `Specify a filename containing [user@]host:port pairs to connect to.`,
		},
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `format, f`,
			Usage: `Specify the output format for data output from subcommands.`,
			Value: `json`,
		},
		cli.StringFlag{
			Name:  `scp-bin`,
			Usage: `Specify the name of the "scp" binary to use for copying files.`,
			Value: `scp`,
		},
		cli.StringFlag{
			Name:  `ssh-bin`,
			Usage: `Specify the name of the "ssh" binary to use for copying files.`,
			Value: `ssh`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		if hosts, err := parseHosts(c); err == nil {
			for _, host := range hosts {
				log.Noticef("HOST: %v", host)
			}
		} else {
			log.Fatal(err)
		}
	}

	app.Run(os.Args)
}

func parseHosts(c *cli.Context) ([]string, error) {
	var data []byte
	var hosts []string

	if hostfile := c.String(`hosts`); hostfile != `` {
		if d, err := fileutil.ReadAll(hostfile); err == nil {
			data = d
		} else {
			return nil, fmt.Errorf("cannot read input: %v", err)
		}
	} else if d, err := ioutil.ReadAll(os.Stdin); err == nil {
		data = d
	} else {
		return nil, fmt.Errorf("cannot read input: %v", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no hosts provided via flag or standard input")
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		if line == `` {
			continue
		} else if strings.HasPrefix(line, `#`) {
			continue
		} else {
			hosts = append(hosts, line)
		}
	}

	return hosts, nil
}

func sshexec(c *cli.Context, hostname string, env map[string]interface{}, script string) (*sshResults, error) {
	scp := c.String(`scp-bin`)
	ssh := c.String(`ssh-bin`)
	flags := make([]string, 0)
	hostname, port := stringutil.SplitPair(hostname, `:`)

	if port == `` {
		port = `22`
	}

	remoteFile := fmt.Sprintf("flak-%d-%d", time.Now().UnixNano(), rand.Intn(65536))

	// have the script remove itself as the final command
	script += fmt.Sprintf("\nret=$?; rm -f '%s'; exit $ret\n", remoteFile)

	flags = append(flags, `-o`, `Port=`+port)

	results := new(sshResults)
	results.StartedAt = time.Now()

	if filename, err := fileutil.WriteTempFile(script, ``); err == nil {
		scpTo := executil.Command(scp, append(flags, filename, fmt.Sprintf("%s:%s", hostname, remoteFile))...)

		for k, v := range env {
			scpTo.SetEnv(k, v)
		}

		scpTo.OnStdout = cmdlog(`scp`, nil)
		scpTo.OnStderr = cmdlog(`scp`, nil)

		if err := scpTo.Run(); err == nil {
			sshTo := executil.Command(ssh, append(flags, hostname, remoteFile)...)

			for k, v := range env {
				sshTo.SetEnv(k, v)
			}

			sshTo.OnStdout = cmdlog(`ssh`, results)
			sshTo.OnStderr = cmdlog(`ssh`, results)

			err := sshTo.Run()
			status := sshTo.WaitStatus()

			results.Status = status.ExitCode
			results.CompletedAt = time.Now()

			return results, err
		} else {
			return nil, fmt.Errorf("script scp: %v", err)
		}
	} else {
		return nil, fmt.Errorf("failed to write script file: %v", err)
	}
}

func cmdlog(tag string, results *sshResults) executil.OutputLineFunc {
	return func(line string, serr bool) {
		lvl := log.INFO

		if rxutil.Match(`(?i)(error|fail|critical|danger)`, line) != nil {
			lvl = log.ERROR
		} else if rxutil.Match(`(?i)(warn|alert)`, line) != nil {
			lvl = log.WARNING
		} else if rxutil.Match(`(?i)(note|notice)`, line) != nil {
			lvl = log.NOTICE
		} else if rxutil.Match(`(?i)(debug)`, line) != nil {
			lvl = log.DEBUG
		}

		log.Logf(lvl, "[%s] %s", tag, line)

		if results != nil {
			if serr {
				results.Stderr = append(results.Stderr, line)
			} else {
				results.Stdout = append(results.Stderr, line)
			}
		}
	}
}
