package main

import (
	"fmt"

	imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/ghetzel/go-stockutil/log"
)

type FolderStats struct {
	UnreadCount int
	RecentCount int
	TotalCount  int
}

func (self *FolderStats) Add(other *FolderStats) {
	if other != nil {
		self.UnreadCount += other.UnreadCount
		self.RecentCount += other.RecentCount
		self.TotalCount += other.TotalCount
	}
}

func (self *FolderStats) String() string {
	return fmt.Sprintf("total:%d\tunread:%d\trecent:%d", self.TotalCount, self.UnreadCount, self.RecentCount)
}

type Folder struct {
	Name       string
	Delimiter  string
	Attributes []string
	profile    *Profile
}

func (self *Folder) String() string {
	return self.Name
}

func (self *Folder) Statistics() (*FolderStats, error) {
	if mbox, err := self.profile.client.Select(self.Name, true); err == nil {
		return &FolderStats{
			UnreadCount: int(mbox.Unseen),
			RecentCount: int(mbox.Recent),
			TotalCount:  int(mbox.Messages),
		}, nil
	} else {
		return nil, fmt.Errorf("Cannot select %q: %v", self.Name, err)
	}
}

func (self *Folder) Expunge() error {
	return self.c().Expunge(nil)
}

func (self *Folder) Mark(flags []Flag, messages ...*Message) error {
	var nativeFlags []interface{}

	for _, f := range flags {
		nativeFlags = append(nativeFlags, f.native())
	}

	seq := new(imap.SeqSet)

	for _, m := range messages {
		log.Debugf("adding %+v", m.Seq())
		seq.AddNum(m.Seq())
	}

	return self.c().Store(
		seq,
		imap.FormatFlagsOp(imap.AddFlags, false),
		nativeFlags,
		nil,
	)
}

func (self *Folder) Delete(messages ...*Message) error {
	return self.Mark([]Flag{
		FlagDeleted,
	}, messages...)
}

func (self *Folder) Messages() <-chan *Message {
	msgchan := make(chan *Message)

	go func() {
		defer close(msgchan)

		if mbox, err := self.c().Select(self.Name, true); err == nil {
			if mbox.Messages > 0 {
				seqset := new(imap.SeqSet)
				seqset.AddRange(1, mbox.Messages)
				items := []imap.FetchItem{
					imap.FetchEnvelope,
					imap.FetchFlags,
				}

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

func (self *Folder) c() *client.Client {
	return self.profile.c()
}
