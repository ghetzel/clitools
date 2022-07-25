package main

import (
	"bytes"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/colorutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type wledLedSet map[int]colorutil.Color

func (self wledLedSet) Has(i int) bool {
	if len(self) == 0 {
		return true
	}

	if _, ok := self[i]; ok {
		return true
	} else {
		return false
	}
}

type wledStreamer struct {
	proto   wledProtocol
	w       io.Writer
	timeout int
	hdr     bool
}

func wled_newStreamer(proto wledProtocol, w io.Writer, timeout int) *wledStreamer {
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

type wledProtocol byte

const (
	wled_WARLS  wledProtocol = 0x1
	wled_DRGB                = 0x2
	wled_DRGBW               = 0x3
	wled_DNRGB               = 0x4
	wled_NOTIFY              = 0x0
)

func (self wledProtocol) WriteBytes(w io.Writer, timeout int, irgb ...byte) {
	var msg = bytes.NewBuffer(nil)

	wled_write(msg, byte(self))
	wled_write(msg, byte(timeout))
	wled_write(msg, irgb...)

	// write to connection in one shot
	wled_write(w, msg.Bytes()...)
}

func (self wledProtocol) WriteTo(w io.Writer, timeout int, i int, r uint8, g uint8, b uint8) {
	self.WriteBytes(w, timeout, []byte{
		byte(i),
		byte(r),
		byte(g),
		byte(b),
	}...)
}

func wled_write(w io.Writer, b ...byte) {
	if _, err := w.Write(b); err != nil {
		log.Fatalf("bad write: %v", err)
	}
}

func parse_wledProtocol(protocol string) wledProtocol {
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

func parse_wledRange(rangespec string) (leds wledLedSet) {
	leds = make(wledLedSet)
	rangespec = strings.TrimSpace(rangespec)

	for _, subrange := range strings.Split(rangespec, `,`) {
		var index, colorspec = stringutil.SplitPairTrimSpace(subrange, `@`)
		var color colorutil.Color

		switch colorspec {
		case ``:
			continue
		case `-`:
			color = colorutil.MustParse(`rgba(0,0,0,1)`)
		default:
			color = colorutil.MustParse(colorspec)
		}

		if a, b := stringutil.SplitPairTrimSpace(index, `:`); a != `` {
			var ai int = typeutil.NInt(a)

			if b != `` {
				var bi int = typeutil.NInt(b)

				for i := ai; i < bi; i++ {
					leds[i] = color
				}
			} else {
				leds[ai] = color
			}
		}
	}

	return
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
			Value:  `debug`,
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
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		var proto wledProtocol = parse_wledProtocol(c.String(`protocol`))
		var addr *net.UDPAddr = parse_wledHost(c.String(`address`))
		var num_leds = c.Int(`led-count`)
		var timeout = c.Int(`timeout`)
		var fx = c.String(`effect`)
		var sleep = c.Duration(`interval`)
		var clearFirst = c.Bool(`clear-first`)

		if conn, err := net.DialUDP(`udp`, nil, addr); err == nil {
			if arg1 := c.Args().First(); arg1 != `-` {
				var ledset wledLedSet = parse_wledRange(strings.Join(c.Args(), `,`))

				if clearFirst {
					var payload = make([]byte, num_leds*4)

					for i := 0; i < num_leds; i++ {
						payload[0+(i*4)] = byte(i)
						payload[1+(i*4)] = 0
						payload[2+(i*4)] = 0
						payload[3+(i*4)] = 0
					}

					proto.WriteBytes(conn, timeout, payload...)
				}

				switch fx {
				case ``, `fill`:
					var r, g, b, _ uint8 = colorutil.MustParse(
						typeutil.OrString(arg1, `#ff0000`),
					).RGBA255()

					var payload = make([]byte, num_leds*4)

					for i := 0; i < num_leds; i++ {
						if !ledset.Has(i) {
							continue
						} else if c := ledset[i]; !c.IsZero() {
							r, g, b, _ = c.RGBA255()
						}

						payload[0+(i*4)] = byte(i)
						payload[1+(i*4)] = r
						payload[2+(i*4)] = g
						payload[3+(i*4)] = b
					}

					proto.WriteBytes(conn, timeout, payload...)
				case `colortrain`:
					var i int = 0
					// var oc colorutil.Color
					var cc colorutil.Color = colorutil.MustParse(`#ff0000`)

					for {
						if !ledset.Has(i) {
							continue
						}

						var r, g, b, _ uint8 = cc.RGBA255()
						proto.WriteTo(conn, timeout, i, r, g, b)
						time.Sleep(sleep)

						if i > 1 {
							// oc = cc
							cc, _ = colorutil.AdjustHue(cc, 1)
							// cc, _ = colorutil.Mix(cc, oc)
						}

						i = (i + 1) % num_leds
					}
				}
			} else {
				var i int = 1

				for {
					var rgb = make([]byte, 3)

					if n, err := os.Stdin.Read(rgb); err == nil && n == 3 {
						proto.WriteBytes(conn, timeout, byte(i), rgb[0], rgb[1], rgb[2])
						time.Sleep(sleep)
					}

					i = (i + 1) % num_leds
				}
			}
		} else {
			log.Fatalf("conn: %v", err)
		}
	}

	app.Run(os.Args)
}
