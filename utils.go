package clitools

import (
	"encoding/json"
	"fmt"
	"os"

	tabwriter "github.com/NonerKao/color-aware-tabwriter"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/go-stockutil/sliceutil"
)

func Print(c *cli.Context, data interface{}, txtfn func()) {
	if data != nil {
		switch c.GlobalString(`format`) {
		case `json`:
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent(``, `  `)
			enc.Encode(data)
		default:
			if txtfn != nil {
				txtfn()
			} else {
				tw := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

				lines := sliceutil.Sliceify(data)
				lines = sliceutil.Compact(lines)

				for _, line := range lines {
					fmt.Fprintf(tw, "%v\n", line)
				}

				tw.Flush()
			}
		}
	}
}
