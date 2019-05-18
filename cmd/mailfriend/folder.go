package main

import (
	imap "github.com/emersion/go-imap"
	"github.com/ghetzel/go-stockutil/log"
)

type Folder struct {
	Name       string
	Delimiter  string
	Attributes []string
	profile    *Profile
}

func (self *Folder) String() string {
	return self.Name
}

func (self *Folder) Messages() <-chan *Message {
	msgchan := make(chan *Message)

	go func() {
		defer close(msgchan)

		if mbox, err := self.profile.client.Select(self.Name, true); err == nil {
			if mbox.Messages > 0 {
				seqset := new(imap.SeqSet)
				seqset.AddRange(1, mbox.Messages)
				items := []imap.FetchItem{imap.FetchEnvelope}

				messages := make(chan *imap.Message)

				go func() {
					if err := self.profile.client.Fetch(seqset, items, messages); err != nil {
						log.Errorf("fetch error: %v", err)
					}
				}()

				for message := range messages {
					msgchan <- &Message{
						message: message,
						folder:  self,
					}
				}
			}
		} else {
			log.Errorf("Cannot select %q: %v", self.Name, err)
		}
	}()

	return msgchan
}
