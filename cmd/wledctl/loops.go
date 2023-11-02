package main

import (
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/stringutil"
)

type LoopControl int

const (
	ControlNothing LoopControl = iota
	ControlBreak
	ControlAdvance
	ControlRetreat
)

type LoopStep string

func (self LoopStep) Parse() (schemes []string, dur time.Duration, ctl LoopControl, perr error) {
	var s = strings.TrimSpace(string(self))
	ctl = ControlNothing

	switch s {
	case `break`:
		ctl = ControlBreak
	case `advance`:
		ctl = ControlAdvance
	case `retreat`:
		ctl = ControlRetreat
	default:
		var schemeset, timespec = stringutil.SplitPairTrimSpace(s, `@`)

		schemes = strings.Split(schemeset, `,`)

		if d, err := time.ParseDuration(timespec); err == nil {
			dur = d
		} else {
			perr = err
			return
		}
	}

	return
}

type LoopConfig []LoopStep
