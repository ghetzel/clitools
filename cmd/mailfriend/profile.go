package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghodss/yaml"
)

var DefaultProfileName = `default`
var ProfileDir = executil.RootOrString(`/etc/mailfriend`, `~/.config/mailfriend`)

type Profile struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
	profile  string
	client   *client.Client
}

func NewProfile(profile string) (*Profile, error) {
	if profile == `` {
		profile = DefaultProfileName
	}

	iclient := &Profile{
		Protocol: `imap`,
		profile:  profile,
	}

	if err := iclient.init(); err == nil {
		return iclient, nil
	} else {
		return nil, fmt.Errorf("Cannot connect to profile %q: %v", iclient.profile, err)
	}
}

func (self *Profile) config() string {
	return fileutil.MustExpandUser(filepath.Join(ProfileDir, fmt.Sprintf("%s.yml", self.profile)))
}

func (self *Profile) init() error {
	if err := self.Load(); err != nil {
		return err
	}

	if err := self.Connect(); err != nil {
		return err
	}

	return nil
}

func (self *Profile) Load() error {
	if data, err := fileutil.ReadAll(self.config()); err == nil {
		return yaml.Unmarshal(data, self)
	} else {
		return err
	}
}

func (self *Profile) Connect() error {
	if err := self.Close(); err != nil {
		return fmt.Errorf("Cannot close existing connection: %v", err)
	}

	var err error

	if self.SSL {
		self.client, err = client.DialTLS(self.Address, nil)
	} else {
		self.client, err = client.Dial(self.Address)
	}

	if err == nil {
		if self.Username != `` && self.Password != `` {
			if err := self.client.Login(self.Username, self.Password); err != nil {
				return err
			}
		}

		return self.client.Noop()
	} else {
		return fmt.Errorf("Cannot connect: %v", err)
	}
}

func (self *Profile) Close() error {
	if self.client != nil {
		if err := self.client.Logout(); err != nil {
			return err
		}

		return self.client.Close()
	}

	return nil
}

func (self *Profile) GetFolder(name string) (*Folder, error) {
	if folders, err := self.ListFolders(name); err == nil {
		switch len(folders) {
		case 1:
			return folders[0], nil
		case 0:
			return nil, fmt.Errorf("Folder %q not found", name)
		default:
			return nil, fmt.Errorf("Too many folders found (expected 1, got %d)", len(folders))
		}
	} else {
		return nil, err
	}
}

func (self *Profile) ListFolders(patterns ...string) ([]*Folder, error) {
	var folders []*Folder
	var wg sync.WaitGroup
	pattern := `*`

	if len(patterns) > 0 && patterns[0] != `` {
		pattern = patterns[0]
	}

	infochan := make(chan *imap.MailboxInfo)

	go func(w *sync.WaitGroup) {
		for info := range infochan {
			folders = append(folders, &Folder{
				Name:       info.Name,
				Delimiter:  info.Delimiter,
				Attributes: info.Attributes,
				profile:    self,
			})
		}

		w.Done()
	}(&wg)

	wg.Add(1)

	if err := self.client.List(``, pattern, infochan); err != nil {
		close(infochan)
		return nil, err
	}

	wg.Wait()

	sort.Slice(folders, func(i int, j int) bool {
		return folders[i].Name < folders[j].Name
	})

	return folders, nil
}
