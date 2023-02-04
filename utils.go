package clitools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tabwriter "github.com/NonerKao/color-aware-tabwriter"
	"github.com/ghetzel/cli"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
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

// Split a slice of strings into a map.  Each element of the slice should take the general form:
//
//   [type:]key[.subkey[.subkey]]]=value
//
// The optional type prefix is parsed using stringutil.ParseType.
func SliceOfKVPairsToMap(pairs []string, joiner string, nester string) map[string]interface{} {
	var out = make(map[string]interface{})

	if joiner == `` {
		joiner = `=`
	}

	if nester == `` {
		nester = `.`
	}

	for _, pair := range pairs {
		var k, v = stringutil.SplitPair(pair, joiner)
		var vT interface{}
		var converted bool
		var typ string

		k, typ = stringutil.SplitPair(k, `:`)

		if tt := stringutil.ParseType(typ); tt != stringutil.Invalid {
			if vv, err := stringutil.ConvertTo(tt, v); err == nil {
				vT = vv
				converted = true
			}
		}

		if !converted {
			vT = stringutil.Autotype(v)
		}

		maputil.DeepSet(out, strings.Split(k, nester), vT)
	}

	return out
}
