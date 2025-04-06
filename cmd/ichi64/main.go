package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bearmini/bitstream-go"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/clitools"
	"github.com/ghetzel/go-stockutil/log"
)

const IChingBase = 0x4dc0
const IChingLen = 64

var IChingEncoding = func() (out [IChingLen]rune) {
	for i := 0; i < IChingLen; i += 1 {
		out[i] = rune(IChingBase + i)
	}

	return
}()

var HtmlStylesheet = func(wrap int) (css string) {
	var lines []string

	lines = append(lines, "html, body, article, span {")

	if wrap > 0 {
		lines = append(lines, fmt.Sprintf("  font-size:  %dvw;", int(100/wrap)))
	}

	lines = append(lines, "  margin:         0;")
	lines = append(lines, "  padding:        0;")
	lines = append(lines, "  overflow:       hidden;")
	lines = append(lines, "  line-height:    0.75em;")
	lines = append(lines, "  letter-spacing: -0.2em;")
	lines = append(lines, "}")

	lines = append(lines, "span {")
	lines = append(lines, "  display:     block;")
	lines = append(lines, "  margin:      0;")
	lines = append(lines, "  white-space: pre-wrap;")
	lines = append(lines, "  word-wrap:   break-word;")
	lines = append(lines, "  width:       auto;")
	lines = append(lines, "  max-width:   100vw;")
	lines = append(lines, "}")

	lines = append(lines, "article {")
	lines = append(lines, "  display:      inline-block;")
	lines = append(lines, "  float:        left;")
	lines = append(lines, "  margin-left:  auto;")
	lines = append(lines, "  margin-right: auto;")
	lines = append(lines, "}")

	css = strings.Join(lines, "\n")
	return
}

func main() {
	app := cli.NewApp()
	app.Name = `ichi64`
	app.Usage = `A base64 encoder that outputs I Ching hexagrams`
	app.Version = clitools.Version

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  `decode, d`,
			Usage: `decode data`,
		},
		cli.BoolFlag{
			Name:  `ignore-garbage, i`,
			Usage: `when decoding, ignore non-I Ching symbols`,
		},
		cli.IntFlag{
			Name:  `wrap, w`,
			Usage: `wrap encoded lines after COLS character (default 76).  Use 0 to disable line wrapping`,
			Value: 76,
		},
		cli.BoolFlag{
			Name:  `html, H`,
			Usage: `output as a printable HTML file`,
		},
	}

	app.Action = func(c *cli.Context) {
		var input io.Reader

		if c.NArg() > 0 {
			input = bytes.NewBufferString(c.Args().Get(0))
		} else {
			input = os.Stdin
		}

		if c.Bool(`decode`) {
			decodeToStdout(c, input)
		} else {
			encodeToIChing(c, input)
		}
	}

	app.Run(os.Args)
}

func encodeToIChing(c *cli.Context, input io.Reader) {
	var inbits = bitstream.NewReader(input, nil)
	var gi int

	print(c, 0)

	for {
		if v, err := inbits.ReadNBitsAsUint8(6); err == nil {
			if gi > 0 && c.Int(`wrap`) > 0 && gi%c.Int(`wrap`) == 0 {
				print(c, '\n')
			}

			print(c, IChingEncoding[int(v)])
			gi += 1
		} else {
			print(c, '\n')
			print(c, 1)
			return
		}
	}
}

func decodeToStdout(c *cli.Context, input io.Reader) {
	var outbits = bitstream.NewWriter(os.Stdout)
	var inscan = bufio.NewReader(input)

	var glyphmap = make(map[rune]uint8)

	for i, ichi := range IChingEncoding {
		glyphmap[ichi] = uint8(i)
	}

	var gi int

	for {
		if r, _, err := inscan.ReadRune(); err == nil {
			if idx, ok := glyphmap[r]; ok {
				if err := outbits.WriteNBitsOfUint8(6, idx); err == nil {
					continue
				} else {
					log.Fatal(err.Error())
				}
			} else if c.Bool(`strict`) {
				log.Fatalf("invalid input character (%v) at position %d", r, gi)
			} else {
				continue
			}

			gi += 1
		} else {
			break
		}
	}
}

func print(c *cli.Context, chr rune) {
	var header bool = (chr == 0)
	var trailer bool = (chr == 1)
	var breaker bool = (chr == '\n')

	if c.Bool(`html`) {
		if header {
			fmt.Printf("<html><head><style>%s</style></head><body><article><span>", HtmlStylesheet(c.Int(`wrap`)))
		} else if trailer {
			fmt.Printf("</span></article></body></html>")
		} else if breaker {
			fmt.Printf("</span><span>")
		}
	} else if breaker {
		fmt.Print("\n")
	}

	if chr >= IChingBase && chr < (IChingBase+IChingLen) {
		fmt.Print(string(chr))
	}
}
