package main

import (
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var PerEffectDefaults = map[string]map[string]interface{}{
	`sequence`: {
		`interval`: typeutil.Duration(`500ms`),
	},
}

type wledStreamer struct {
	proto   Protocol
	w       io.Writer
	timeout int
	hdr     bool
}

func wled_newStreamer(proto Protocol, w io.Writer, timeout int) *wledStreamer {
	return &wledStreamer{
		proto:   proto,
		w:       w,
		timeout: timeout,
	}
}

func (self *wledStreamer) Write(b []byte) (int, error) {
	if !self.hdr {
		self.w.Write([]byte{byte(self.proto)})
		self.w.Write([]byte{byte(self.timeout)})
		self.hdr = true
	}

	return self.w.Write(b)
}

func wled_write(w io.Writer, b ...byte) {
	if _, err := w.Write(b); err != nil {
		log.Fatalf("bad write: %v", err)
	}
}

func parse_wledProtocol(protocol string) Protocol {
	var p = protocol
	p = strings.ToLower(p)
	p = strings.TrimSpace(p)

	switch p {
	case `warls`, `1`:
		return wled_WARLS
	case `drgb`, `2`:
		return wled_DRGB
	case `drgbw`, `3`:
		return wled_DRGBW
	case `dnrgb`, `4`:
		return wled_DNRGB
	case `notify`, `5`:
		return wled_NOTIFY
	default:
		log.Fatalf("unknown protocol %q", protocol)
		return 0
	}
}

func parse_wledHost(addr string) *net.UDPAddr {
	if udpaddr, err := net.ResolveUDPAddr(`udp`, addr); err == nil {
		return udpaddr
	} else {
		log.Fatalf("bad host: %v", err)
		return nil
	}
}

func main() {
	var app = cli.NewApp()
	app.Name = `wledctl`
	app.Usage = `command line utility for controlling ESP8266 LED devices using WLED over UDP`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:   `address, a`,
			Usage:  `The WLED IP[:PORT] to communicate with`,
			Value:  `127.0.0.1:21324`,
			EnvVar: `WLEDCTL_HOST`,
		},
		cli.IntFlag{
			Name:   `led-count, n`,
			Usage:  `The number of LEDs being addressed.`,
			Value:  60,
			EnvVar: `WLEDCTL_LED_COUNT`,
		},
		// cli.IntFlag{
		// 	Name:   `brightness, b`,
		// 	Usage:  `How bright the LEDs should be (0-255)`,
		// 	Value:  128,
		// 	EnvVar: `WLEDCTL_BRIGHTNESS`,
		// },
		cli.StringFlag{
			Name:   `scheme, s`,
			Usage:  `Specify a name to save the given color scheme as, or if no other arguments are given, the scheme to apply.`,
			EnvVar: `WLEDCTL_SCHEME`,
		},
		cli.StringFlag{
			Name:   `config, c`,
			Usage:  `The configuration file containing named effects.`,
			EnvVar: `WLEDCTL_CONFIG_FILE`,
			Value:  DefaultConfigName,
		},
		cli.StringFlag{
			Name:   `protocol, p`,
			Usage:  `Specify the WLED protocol to use: WARLS (1) DRGB (2) DRGBW (3) DNRBG (4) NOTIFIER (0)`,
			Value:  `warls`,
			EnvVar: `WLEDCTL_PROTOCOL`,
		},
		cli.IntFlag{
			Name:  `timeout, t`,
			Usage: `How many seconds to wait before resuming normal light mode.`,
			Value: 255,
		},
		cli.DurationFlag{
			Name:  `interval, i`,
			Usage: `The frame interval between successive updates.`,
			Value: 100 * time.Millisecond,
		},
		cli.StringFlag{
			Name:  `effect, x`,
			Usage: `The named of the pre-defined effect to trigger`,
		},
		cli.BoolFlag{
			Name:  `clear-first, C`,
			Usage: `Whether to clear all LEDs before changing state.`,
		},
		cli.DurationFlag{
			Name:  `transition-duration, T`,
			Usage: `How long the effect should take to complete`,
			Value: DefaultTransitionShaderDuration,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))

		return nil
	}

	app.Action = func(c *cli.Context) {
		// var proto Protocol = parse_wledProtocol(c.String(`protocol`))
		var addr *net.UDPAddr = parse_wledHost(c.String(`address`))
		var num_leds = c.Int(`led-count`)
		var timeout = c.Int(`timeout`)
		var fx = c.String(`effect`)
		var sleep = c.Duration(`interval`)
		var clearFirst = c.Bool(`clear-first`)
		var effectTime = c.Duration(`transition-duration`)
		var configName = c.String(`config`)
		var cfg *Config

		if c, err := LoadConfig(configName); err == nil {
			cfg = c
		} else {
			log.Fatalf("load config: %v", err)
		}

		// if args := sliceutil.CompactString(c.Args()); schemeName != `` && len(args) > 0 {
		// 	cfg.Schemes[schemeName] = args
		// }

		// if err := SaveConfig(configName, cfg); err == nil {
		// 	log.Debugf("updated config: %v", configName)
		// } else {
		// 	log.Fatalf("save config: %v", err)
		// }

		if conn, err := net.DialUDP(`udp`, nil, addr); err == nil {
			var strip = NewDisplay(conn, num_leds)
			// var ledset LEDSet = parse_wledRange(strings.Join(c.Args(), `,`))

			if kv, ok := PerEffectDefaults[fx]; ok && len(kv) > 0 {
				for k, v := range kv {
					if typeutil.IsEmpty(v) {
						continue
					}

					switch k {
					case `interval`:
						sleep = typeutil.Duration(v)
					case `timeout`:
						timeout = typeutil.NInt(v)
					}
				}
			}

			strip.FrameInterval = sleep
			strip.AutoclearTimeout = uint8(timeout)
			strip.ClearFirst = clearFirst
			strip.TransitionShaderDuration = effectTime

			var schemes []string = c.Args()

			// if sn := cfg.Scheme(schemeName); len(sn) > 0 {
			// 	schemes = sn
			// }

			schemes = sliceutil.CompactString(schemes)

			if len(schemes) == 0 {
				schemes = []string{`black`}
			}

			for i := 0; i < len(schemes); i++ {
				var scheme = schemes[i]
				switch scheme[0] {
				case '@':
					if len(cfg.Loops) > 0 {
						if loop := cfg.Loops[scheme[1:]]; len(loop) > 0 {
						ForeverLoop:
							for {
								for _, step := range loop {
									if schemes, dur, ctl, err := step.Parse(); err == nil {
										switch ctl {
										case ControlBreak:
											break ForeverLoop
										default:
											cfg.ApplyScheme(strip, schemes, fx)
											time.Sleep(dur)
										}
									} else {
										log.Fatal(err)
									}
								}
							}
						}
					}
				default:
					cfg.ApplyScheme(strip, schemes[i:], fx)
				}
			}

		} else {
			log.Fatalf("conn: %v", err)
		}
	}

	app.Run(os.Args)
}
