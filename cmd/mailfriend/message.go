package main

import (
	"encoding/json"
	"time"

	imap "github.com/emersion/go-imap"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
)

type Message struct {
	message *imap.Message
	folder  *Folder
}

func (self *Message) hdr() *imap.Envelope {
	return self.message.Envelope
}

func (self *Message) Subject() string {
	if subject := self.hdr().Subject; subject != `` {
		return subject
	} else {
		return `(no subject)`
	}
}

func (self *Message) Timestamp() time.Time {
	return self.hdr().Date
}

func (self *Message) From() *Contact {
	if from := self.hdr().From; len(from) > 0 {
		return &Contact{
			Name:    from[0].PersonalName,
			Address: from[0].MailboxName,
			Domain:  from[0].HostName,
		}
	} else {
		return &Contact{
			Address: `unknown`,
		}
	}
}

func (self *Message) addrs(header ContactSource) (contacts []*Contact) {
	var addrs []*imap.Address

	switch header {
	case Cc:
		addrs = self.hdr().Cc
	case Bcc:
		addrs = self.hdr().Bcc
	case ReplyTo:
		addrs = self.hdr().ReplyTo
	default:
		addrs = self.hdr().To
	}

	for _, addr := range addrs {
		contacts = append(contacts, &Contact{
			Name:    addr.PersonalName,
			Address: addr.MailboxName,
			Domain:  addr.HostName,
		})
	}

	return
}

func (self *Message) To() []*Contact {
	return self.addrs(To)
}

func (self *Message) Cc() []*Contact {
	return self.addrs(Cc)
}

func (self *Message) Bcc() []*Contact {
	return self.addrs(Bcc)
}

func (self *Message) ReplyTo() []*Contact {
	return self.addrs(ReplyTo)
}

func (self *Message) Recipients() (contacts []*Contact) {
	contacts = append(contacts, self.To()...)
	contacts = append(contacts, self.Cc()...)
	contacts = append(contacts, self.Bcc()...)
	return
}

func (self *Message) ID() string {
	return self.hdr().MessageId
}

func (self *Message) ParentID() string {
	return self.hdr().InReplyTo
}

func (self *Message) Flags() []Flag {
	return ParseFlags(self.message.Flags)
}

func (self *Message) IsRead() bool {
	return sliceutil.Contains(self.Flags(), FlagRead)
}

func (self *Message) IsStarred() bool {
	return sliceutil.Contains(self.Flags(), FlagStarred)
}

func (self *Message) String() string {
	var line string
	var tokens string

	if self.IsStarred() {
		tokens += "*"
	} else {
		tokens += " "
	}

	if !self.IsRead() {
		tokens += "!"
	} else {
		tokens += " "
	}

	line += log.CSprintf(
		"${yellow}[%s]${reset}${green}[%v]${reset}\t%v\t${blue}%v${reset}",
		tokens,
		self.Timestamp().Format(`2006-01-02 15:04:05`),
		self.Subject(),
		self.From(),
	)

	return line
}

func (self *Message) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{
		`ID`:        self.ID(),
		`Subject`:   self.Subject(),
		`Timestamp`: self.Timestamp(),
		`From`:      self.From().String(),
	}

	if flags := ParseFlags(self.message.Flags); len(flags) > 0 {
		out[`Flags`] = sliceutil.Stringify(flags)
	}

	if v := self.ParentID(); v != `` {
		out[`ParentID`] = v
	}

	if v := self.To(); len(v) > 0 {
		out[`To`] = sliceutil.Stringify(v)
	}

	if v := self.Cc(); len(v) > 0 {
		out[`Cc`] = sliceutil.Stringify(v)
	}

	if v := self.Bcc(); len(v) > 0 {
		out[`Bcc`] = sliceutil.Stringify(v)
	}

	if v := self.ReplyTo(); len(v) > 0 {
		out[`ReplyTo`] = sliceutil.Stringify(v)
	}

	return json.Marshal(out)
}
