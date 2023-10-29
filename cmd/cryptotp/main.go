package main

import (
	"fmt"
	"os"

	"gorthub.com/ghetzel/cli"
	"gorthub.com/ghetzel/clitools"
	"gorthub.com/ghetzel/go-stockutil/executil"
	"gorthub.com/ghetzel/go-stockutil/log"
)

func main() {
	var app = cli.NewApp()
	var cryptotp *Config

	app.Name = `cryptotp`
	app.Usage = `TOTP-based symmetric encryption/decryption utility.`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:   `config, c`,
			Usage:  `Location of the encrypted password store.`,
			Value:  executil.RootOrString(`/etc/cryptotp.cfg`, `~/.config/cryptotp/config.cfg`),
			EnvVar: `CRYPTOTP_CONFIG`,
		},
		cli.StringFlag{
			Name:   `config-key, C`,
			Usage:  `The secret key used to decrypt and encrypt the configuration.`,
			EnvVar: `CRYPTOTP_CONFIG_KEY`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))

		if cfg, err := LoadConfig(
			c.GlobalString(`config`),
			c.GlobalString(`config-key`),
		); err == nil {
			cryptotp = cfg
		} else {
			return err
		}

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  `create`,
			Usage: `Create a new TOTP token.`,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   `issuer, i`,
					Usage:  `The name of the issuing website or entity.`,
					EnvVar: `CRYPTOTP_DEFAULT_ISSUER`,
				},
				cli.StringFlag{
					Name:   `account, a`,
					Usage:  `The name account on the issuer.`,
					EnvVar: `CRYPTOTP_DEFAULT_ACCOUNT`,
				},
			},
			Action: func(c *cli.Context) {
				if secret, err := GenerateSecret(
					c.String(`issuer`),
					c.String(`account`),
					cryptotp,
				); err == nil {
					fmt.Println(secret)
				} else {
					log.Fatal(err)
				}
			},
		},
		{
			Name:  `dump`,
			Usage: `Dump all configured secret keys and their current valid TOTP token.`,
			Flags: []cli.Flag{},
			Action: func(c *cli.Context) {
				for _, secret := range cryptotp.Secrets {
					fmt.Printf("%v %s\n", secret.String(), secret.Code())
				}
			},
		},
	}

	// app.Action = func(c *cli.Context) {}

	app.Run(os.Args)
}
