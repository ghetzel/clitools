package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
	"golang.org/x/net/html"
)

func main() {
	app := cli.NewApp()
	app.Name = `xq`
	app.Usage = `Like jq, but for HTML/XML.`
	app.ArgsUsage = `EXPRESSION`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `warning`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `format, f`,
			Usage: `Specify the output format for data output from subcommands.`,
			Value: `json`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		var input io.Reader = os.Stdin

		if input != nil {
			if doc, err := htmldoc(os.Stdin); err == nil {
				elements := make([]map[string]interface{}, 0)

				if len(c.Args()) == 0 {
					if out, err := doc.Html(); err == nil {
						fmt.Println(string(out))
					} else {
						log.Fatalf("error formatting HTML: %v", err)
					}
				} else {
					doc.Find(strings.Join(c.Args(), ` `)).Each(func(i int, match *goquery.Selection) {
						if len(match.Nodes) > 0 {
							for _, node := range match.Nodes {
								if nodeData := htmlNodeToMap(node); len(nodeData) > 0 {
									elements = append(elements, nodeData)
								}
							}
						}
					})

					clitools.Print(c, elements, nil)
				}
			} else {
				log.Fatalf("Cannot parse document: %v", err)
			}
		}
	}

	app.Run(os.Args)
}

func htmldoc(docI interface{}) (*goquery.Document, error) {
	if d, ok := docI.(*goquery.Document); ok {
		return d, nil
	} else if d, ok := docI.(string); ok {
		return goquery.NewDocumentFromReader(bytes.NewBufferString(d))
	} else if d, ok := docI.(io.Reader); ok {
		return goquery.NewDocumentFromReader(d)
	} else if d, ok := docI.(template.HTML); ok {
		return goquery.NewDocumentFromReader(bytes.NewBufferString(string(d)))
	} else {
		return nil, fmt.Errorf("Expected a HTML document string or object, got: %T", docI)
	}
}

func htmlNodeToMap(node *html.Node) map[string]interface{} {
	output := make(map[string]interface{})

	if node != nil && node.Type == html.ElementNode {
		text := ``
		children := make([]map[string]interface{}, 0)
		attrs := make(map[string]interface{})

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			switch child.Type {
			case html.TextNode:
				text += child.Data
			case html.ElementNode:
				if child != node {
					if childData := htmlNodeToMap(child); len(childData) > 0 {
						children = append(children, childData)
					}
				}
			}
		}

		text = strings.TrimSpace(text)

		for _, attr := range node.Attr {
			attrs[attr.Key] = stringutil.Autotype(attr.Val)
		}

		if len(attrs) > 0 {
			output[`attributes`] = attrs
		}

		if text != `` {
			output[`text`] = text
		}

		if len(children) > 0 {
			output[`children`] = children
		}

		// only if the node has anything useful at all in it...
		if len(output) > 0 {
			output[`name`] = node.DataAtom.String()
		}
	}

	return output
}
