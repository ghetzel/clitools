package main

import (
	"time"

	imap "github.com/emersion/go-imap"
	"github.com/ghetzel/go-stockutil/log"
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

func (self *Message) to(header string) (contacts []*Contact) {
	var addrs []*imap.Address

	switch header {
	case `cc`:
		addrs = self.hdr().Cc
	case `bcc`:
		addrs = self.hdr().Bcc
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
	return self.to(`to`)
}

func (self *Message) Cc() []*Contact {
	return self.to(`cc`)
}

func (self *Message) Bcc() []*Contact {
	return self.to(`bcc`)
}

func (self *Message) Recipients() (contacts []*Contact) {
	contacts = append(contacts, self.To()...)
	contacts = append(contacts, self.Cc()...)
	contacts = append(contacts, self.Bcc()...)
	return
}

func (self *Message) String() string {
	return log.CSprintf("${green}[%v]${reset}\t%v\t${blue}%v${reset}", self.Timestamp().Format(`2006-01-02 15:04:05`), self.Subject(), self.From())
}
