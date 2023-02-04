package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/tarm/serial"
)

func main() {
	app := cli.NewApp()
	app.Name = `prosign`
	app.Usage = `A utility to control ProLite M2014-R public display marquee signage`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `device, d`,
			Usage: `Path to the serial port used to communicate with sign(s).`,
			Value: `/dev/cu.usbserial-130`,
		},
		cli.IntFlag{
			Name:  `baud, b`,
			Usage: `The serial baud rate`,
			Value: 9600,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		var prolite = &serial.Config{
			Name:        c.String(`device`),
			Baud:        c.Int(`baud`),
			ReadTimeout: time.Second,
		}

		if port, err := serial.OpenPort(prolite); err == nil {
			var resp = make([]byte, 128)
			var _, err = port.Write([]byte("<ID01>\r\n"))
			log.FatalIf(err)

			_, err = port.Read(resp)
			log.FatalIf(err)
			fmt.Printf("%q", resp[:])
		} else {
			log.Fatal(err)
		}

	}

	app.Run(os.Args)
}
