package main

import (
	"encoding/xml"

	"github.com/ghetzel/go-stockutil/httputil"
)

var OfxHomeURL = `http://www.ofxhome.com`

func PopulateFromOfxHome(into interface{}, ohid int) error {
	if client, err := httputil.NewClient(OfxHomeURL); err == nil {
		if res, err := client.Get(`/api.php`, map[string]interface{}{
			`lookup`: ohid,
		}, nil); err == nil {
			defer res.Body.Close()
			return xml.NewDecoder(res.Body).Decode(into)
		} else {
			return err
		}
	} else {
		return err
	}
}
