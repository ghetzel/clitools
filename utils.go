package clitools

import (
	"encoding/json"
	"fmt"
	"os"

	tabwriter "github.com/NonerKao/color-aware-tabwriter"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func Print(c *cli.Context, data interface{}, txtfn func()) {
	if data != nil && ((typeutil.IsArray(data) || typeutil.IsMap(data)) && sliceutil.Len(data) > 0) {
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
