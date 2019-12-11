package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/jroimartin/gocui"
)

type hostOutput struct {
	Hostname    string
	Line        string
	Level       log.Level
	Stderr      bool
	Error       error
	ExitCode    int
	StartedAt   time.Time
	CompletedAt time.Time
}

func (self *hostOutput) Duration() time.Duration {
	return self.CompletedAt.Sub(self.StartedAt)
}

func (self *hostOutput) String() string {
	if self.Error != nil {
		return self.Error.Error()
	} else {
		return fmt.Sprintf("exited status %d in %v", self.ExitCode, self.Duration().Round(time.Millisecond))
	}
}

const PerHostHistory int = 1024

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
			Value:  `notice`,
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
		cli.StringFlag{
			Name:  `ssh-config-file, F`,
			Usage: `Specify the SSH configuration file to use.`,
		},
		cli.DurationFlag{
			Name:  `connect-timeout, t`,
			Usage: `Specify the connection timeout for each SSH connection.`,
			Value: 10 * time.Second,
		},
		cli.StringSliceFlag{
			Name:  `ssh-option, o`,
			Usage: `Specify an SSH command line option in the form "-o Key=Value"`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		if c.NArg() == 0 {
			log.Fatalf("Must specify a command or @filename to run")
			return
		}

		if hosts, err := parseHosts(c); err == nil {
			if len(hosts) == 0 {
				log.Warningf("no hosts provided")
				return
			}

			var wg sync.WaitGroup
			var script string
			var multiline bool

			if scriptfile := c.Args().First(); strings.HasPrefix(scriptfile, `@`) {
				scriptfile = strings.TrimPrefix(scriptfile, `@`)

				if data, err := fileutil.ReadAll(scriptfile); err == nil {
					script = string(data)
					multiline = true
				} else {
					log.Fatalf("failed to read %q: %v", scriptfile, err)
					return
				}
			} else {
				script = strings.Join(c.Args(), ` `)
			}

			script = strings.TrimSpace(script)

			if script == `` {
				log.Fatalf("Must specify a command or @filename to run")
				return
			}

			if gui, err := gocui.NewGui(gocui.OutputNormal); err == nil {
				defer gui.Close()

				sort.Strings(hosts)
				hosts = sliceutil.UniqueStrings(hosts)
				// buffers := make(map[string]*LineBuffer)
				buffers := make(map[string]*bytes.Buffer)
				currentHost := hosts[0]

				gui.SetManagerFunc(func(g *gocui.Gui) error {
					maxX, maxY := g.Size()

					if side, err := g.SetView(`side`, -1, -1, int(0.2*float32(maxX)), maxY-5); err == nil {
						for _, host := range hosts {
							fmt.Fprintf(side, "%s\n", host)
						}
					} else if err != gocui.ErrUnknownView {
						return err
					}

					if logs, err := g.SetView(`logs`, int(0.2*float32(maxX)), -1, maxX, maxY-5); err == nil {
						outchan := make(chan *hostOutput, 0)

						refreshCurrentHost := func() {
							logs.Clear()

							if buf, ok := buffers[currentHost]; ok {
								io.Copy(logs, buf)
								// if _, rows := logs.Size(); rows > 0 {
								// 	lines := buf.Lines()

								// 	// if rows < len(lines) {
								// 	// 	lines = lines[len(lines)-rows:]
								// 	// }

								// 	for _, line := range lines {
								// 		fmt.Fprintf(logs, "%s\n", line)
								// 	}
								// }
							}
						}

						go func() {
							for out := range outchan {
								if buf, ok := buffers[out.Hostname]; ok && buf != nil {
									if out.CompletedAt.IsZero() {
										buf.WriteString(out.Line + "\n")

										if out.Hostname == currentHost {
											refreshCurrentHost()
										}
									} else {
										buf.WriteString(fmt.Sprintf("COMPLETED: %v\n", out))
									}
								}
							}
						}()

						for _, host := range hosts {
							// buffers[host] = NewLineBuffer(PerHostHistory)
							buffers[host] = bytes.NewBuffer(nil)
							wg.Add(1)
							go sshexec(&wg, outchan, c, host, nil, script, multiline)
						}

						refreshCurrentHost()
						wg.Wait()
					} else if err != gocui.ErrUnknownView {
						return err
					}

					if cmdline, err := g.SetView(`cmdline`, -1, maxY-5, maxX, maxY); err == nil {
						fmt.Fprintf(cmdline, "TEST")
					} else if err != gocui.ErrUnknownView {
						return err
					}

					return nil
				})

				if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
					return gocui.ErrQuit
				}); err != nil {
					log.Fatalf("quit")
				}

				switch err := gui.MainLoop(); err {
				case nil:
					fallthrough
				case gocui.ErrQuit:
					log.Noticef("Quitting....")
					return
				default:
					log.Fatalf("mainloop: %v", err)
				}
			} else {
				log.Fatalf("gui: %v", err)
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
	var lines []string

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

	if ifs := os.Getenv(`IFS`); ifs != `` {
		lines = strings.Split(string(data), ifs)
	} else {
		lines = rxutil.Whitespace.Split(string(data), -1)
	}

	for _, line := range lines {
		if line := strings.TrimSpace(line); line == `` {
			continue
		} else if strings.HasPrefix(line, `#`) {
			continue
		} else {
			hosts = append(hosts, line)
		}
	}

	return hosts, nil
}

func sshexec(
	wg *sync.WaitGroup,
	outchan chan *hostOutput,
	c *cli.Context,
	hostname string,
	env map[string]interface{},
	script string,
	multiline bool,
) {
	final := new(hostOutput)

	defer func() {
		wg.Done()
		outchan <- final
	}()

	var sshTo *executil.Cmd

	scp := c.String(`scp-bin`)
	ssh := c.String(`ssh-bin`)
	flags := make([]string, 0)
	hostname, port := stringutil.SplitPair(hostname, `:`)
	remoteFile := ``

	if multiline {
		remoteFile = fmt.Sprintf("flak-%d-%d", time.Now().UnixNano(), rand.Intn(65536))

		// have the script remove itself as the final command
		script += fmt.Sprintf("\nret=$?; rm -f '%s'; exit $ret\n", remoteFile)
	}

	if port != `` {
		flags = append(flags, `-q`, `-o`, `Port=`+port)
	}

	flags = append(flags, `-q`, `-o`, `BatchMode=yes`)

	if cf := c.String(`ssh-config-file`); cf != `` {
		flags = append(flags, `-F`, fileutil.MustExpandUser(cf))
	}

	if ct := c.Duration(`connect-timeout`).Round(time.Second); ct > 0 {
		flags = append(flags, `-o`, fmt.Sprintf("ConnectTimeout=%d", ct/time.Second))
	}

	for _, pair := range c.StringSlice(`ssh-option`) {
		pair = strings.TrimSpace(pair)

		if pair != `` {
			flags = append(flags, `-o`, pair)
		}
	}

	final.StartedAt = time.Now()
	final.Hostname = hostname

	if multiline {
		if filename, err := fileutil.WriteTempFile(script, ``); err == nil {
			scpTo := executil.Command(scp, append(flags, filename, fmt.Sprintf("%s:%s", hostname, remoteFile))...)

			for k, v := range env {
				scpTo.SetEnv(k, v)
			}

			scpTo.OnStdout = cmdlog(hostname, nil)
			scpTo.OnStderr = cmdlog(hostname, nil)

			log.Debugf("[%s] exec: %s", hostname, strings.Join(scpTo.Args, ` `))

			if err := scpTo.Run(); err == nil {

			} else {
				final.Error = fmt.Errorf("script scp: %v", err)
				return
			}
		} else {
			final.Error = fmt.Errorf("failed to write script file: %v", err)
			return
		}

		sshTo = executil.Command(ssh, append(flags, hostname, `./`+remoteFile)...)
	} else {
		sshTo = executil.Command(ssh, append(flags, hostname, script)...)
	}

	for k, v := range env {
		sshTo.SetEnv(k, v)
	}

	sshTo.OnStdout = cmdlog(hostname, outchan)
	sshTo.OnStderr = cmdlog(hostname, outchan)

	log.Debugf("[%s] exec: %s", hostname, strings.Join(sshTo.Args, ` `))

	final.Error = sshTo.Run()

	status := sshTo.WaitStatus()
	final.ExitCode = status.ExitCode
	final.CompletedAt = time.Now()

	return
}

func cmdlog(hostname string, outchan chan *hostOutput) executil.OutputLineFunc {
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

		if outchan != nil {
			if serr {
				outchan <- &hostOutput{
					Hostname: hostname,
					Stderr:   true,
					Line:     line,
					Level:    lvl,
				}
			} else {
				outchan <- &hostOutput{
					Hostname: hostname,
					Stderr:   false,
					Line:     line,
					Level:    lvl,
				}
			}
		}
	}
}
