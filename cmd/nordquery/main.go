package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
)

var nordqueryCacheLocation = fileutil.MustExpandUser(`~/.cache/nordquery-servers.json`)

type nordServerCategory struct {
	Name string `json:"name"`
}

type nordServerLocation struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"long"`
}

type nordServerFeatures struct {
	IKEv2               bool `json:"ikev2"`
	IKEv2v6             bool `json:"ikev2_v6"`
	L2TP                bool `json:"l2tp"`
	OpenVpnDedicatedTCP bool `json:"openvpn_dedicated_tcp"`
	OpenVpnDedicatedUDP bool `json:"openvpn_dedicated_udp"`
	OpenVpnTCP          bool `json:"openvpn_tcp"`
	OpenVpnTCPTLSCrypt  bool `json:"openvpn_tcp_tls_crypt"`
	OpenVpnTCPV6        bool `json:"openvpn_tcp_v6"`
	OpenVpnUDP          bool `json:"openvpn_udp"`
	OpenVpnUDPTLSCrypt  bool `json:"openvpn_udp_tls_crypt"`
	OpenVpnUDPV6        bool `json:"openvpn_udp_v6"`
	OpenVpnXorTCP       bool `json:"openvpn_xor_tcp"`
	OpenVpnXorUDP       bool `json:"openvpn_xor_udp"`
	PPTP                bool `json:"pptp"`
	Proxy               bool `json:"proxy"`
	ProxyCybersec       bool `json:"proxy_cybersec"`
	ProxySSL            bool `json:"proxy_ssl"`
	ProxySSLCybersec    bool `json:"proxy_ssl_cybersec"`
	Skylark             bool `json:"skylark"`
	Socks               bool `json:"socks"`
	WireguardUDP        bool `json:"wireguard_udp"`
}

type nordServer struct {
	ID         int                  `json:"id"`
	Address    string               `json:"ip_address"`
	Keywords   []string             `json:"search_keywords"`
	Categories []nordServerCategory `json:"categories"`
	Name       string               `json:"name"`
	Domain     string               `json:"domain"`
	Price      float64              `json:"price"`
	Flag       string               `json:"flag"`
	Country    string               `json:"country"`
	Location   nordServerLocation   `json:"location"`
	Load       int                  `json:"load"`
	Features   nordServerFeatures   `json:"features"`
}

func (self *nordServer) HasFeature(name string) bool {
	switch name {
	case "ikev2":
		return self.Features.IKEv2
	case "ikev2_v6":
		return self.Features.IKEv2v6
	case "l2tp":
		return self.Features.L2TP
	case "openvpn_dedicated_tcp":
		return self.Features.OpenVpnDedicatedTCP
	case "openvpn_dedicated_udp":
		return self.Features.OpenVpnDedicatedUDP
	case "openvpn_tcp":
		return self.Features.OpenVpnTCP
	case "openvpn_tcp_tls_crypt":
		return self.Features.OpenVpnTCPTLSCrypt
	case "openvpn_tcp_v6":
		return self.Features.OpenVpnTCPV6
	case "openvpn_udp":
		return self.Features.OpenVpnUDP
	case "openvpn_udp_tls_crypt":
		return self.Features.OpenVpnUDPTLSCrypt
	case "openvpn_udp_v6":
		return self.Features.OpenVpnUDPV6
	case "openvpn_xor_tcp":
		return self.Features.OpenVpnXorTCP
	case "openvpn_xor_udp":
		return self.Features.OpenVpnXorUDP
	case "pptp":
		return self.Features.PPTP
	case "proxy":
		return self.Features.Proxy
	case "proxy_cybersec":
		return self.Features.ProxyCybersec
	case "proxy_ssl":
		return self.Features.ProxySSL
	case "proxy_ssl_cybersec":
		return self.Features.ProxySSLCybersec
	case "skylark":
		return self.Features.Skylark
	case "socks", "socks5":
		return self.Features.Socks
	case "wg", "wireguard", "wireguard_udp":
		return self.Features.WireguardUDP
	}

	return false
}

func main() {
	app := cli.NewApp()
	app.Name = `nordquery`
	app.Usage = `NordVPN server query tool.`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `debug`,
			EnvVar: `LOGLEVEL`,
		},
		cli.BoolFlag{
			Name:  `sync, s`,
			Usage: `Whether to refresh the local NordVPN server cache from the remote URL.`,
		},
		cli.StringFlag{
			Name:   `server-list-url, S`,
			Usage:  `Specify the URL where the list of active NordVPN servers lives.`,
			Value:  `https://nordvpn.com/api/server`,
			EnvVar: `NORDQUERY_URL`,
		},
		cli.StringSliceFlag{
			Name:  `feature, F`,
			Usage: `Specifies a server feature that must be present in the output.`,
		},
		cli.StringFlag{
			Name:  `format, f`,
			Usage: `Specify the output format.`,
		},
		cli.StringFlag{
			Name:  `country, c`,
			Usage: `Specify the country to filter results by.`,
		},
		cli.BoolFlag{
			Name:  `sort-load, l`,
			Usage: `Sorts the matching servers by load (ascending)`,
		},
		cli.IntFlag{
			Name:  `number, n`,
			Usage: `How many servers to return.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		if c.Bool(`sync`) {
			log.FatalIf(syncNordList(c.String(`server-list-url`)))
		}

		if file, err := os.Open(nordqueryCacheLocation); err == nil {
			defer file.Close()

			var list []*nordServer
			var match []*nordServer

			log.FatalIf(json.NewDecoder(file).Decode(&list))

		ServerLoop:
			for _, server := range list {
				for _, feat := range c.StringSlice(`feature`) {
					if !server.HasFeature(feat) {
						continue ServerLoop
					}
				}

				if country := c.String(`country`); country != `` {
					if strings.ToLower(server.Flag) != strings.ToLower(country) {
						continue ServerLoop
					}
				}

				match = append(match, server)
			}

			if c.Bool(`sort-load`) {
				sort.Slice(match, func(i int, j int) bool {
					return match[i].Load < match[j].Load
				})
			}

			if n := c.Int(`number`); n > 0 && n < len(match) {
				match = match[:n]
			}

			switch c.String(`format`) {
			case ``, `json`:
				json.NewEncoder(os.Stdout).Encode(&match)
			default:
				for _, server := range match {
					switch c.String(`format`) {
					case `domain`:
						fmt.Println(server.Domain)
					case `ip`:
						fmt.Println(server.Address)
					}
				}
			}
		} else {
			log.Fatal(err)
		}
	}

	app.Run(os.Args)
}

func syncNordList(url string) error {
	if res, err := http.Get(url); err == nil {
		if res.StatusCode >= 400 {
			return fmt.Errorf("failed to retrieve server list: %v", err)
		}

		_, err := fileutil.WriteFile(res.Body, nordqueryCacheLocation)
		return err
	} else {
		return err
	}
}
