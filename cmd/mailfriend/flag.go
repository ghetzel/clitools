package main

import (
	imap "github.com/emersion/go-imap"
)

type Flag int

const (
	FlagRead Flag = iota
	FlagAnswered
	FlagStarred
	FlagDeleted
	FlagDraft
	FlagRecent
)

func (self Flag) String() string {
	switch self {
	case FlagRead:
		return `read`
	case FlagAnswered:
		return `answered`
	case FlagStarred:
		return `starred`
	case FlagDeleted:
		return `deleted`
	case FlagDraft:
		return `draft`
	case FlagRecent:
		return `recent`
	default:
		return `unknown`
	}
}

func ParseFlag(flag interface{}) Flag {
	if v, ok := flag.(Flag); ok {
		return v
	} else if v, ok := flag.(string); ok {
		switch v {
		case imap.SeenFlag:
			return FlagRead
		case imap.AnsweredFlag:
			return FlagAnswered
		case imap.FlaggedFlag:
			return FlagStarred
		case imap.DeletedFlag:
			return FlagDeleted
		case imap.DraftFlag:
			return FlagDraft
		case imap.RecentFlag:
			return FlagRecent
		}
	}

	return Flag(-1)
}

func ParseFlags(flags []string) []Flag {
	out := make([]Flag, 0)

	for _, flag := range flags {
		if f := ParseFlag(flag); f >= FlagRead {
			out = append(out, f)
		}
	}

	return out
}
