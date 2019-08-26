package main

import (
	"crypto/rand"
	"fmt"

	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/pivot/v3"
)

var DefaultDatabase = executil.RootOrString(`sqlite://var/db/ofxtool.db`, `sqlite://~/.config/ofxtool/ofxtool.db`)
var PrivateKeyPath = executil.RootOrString(`/etc/ofxtool/default.key`, `~/.config/ofxtool/default.key`)
var PrivateKey [32]byte

func ValidatePrivateKey() error {
	for _, b := range PrivateKey {
		if b != 0 {
			return nil
		}
	}

	return fmt.Errorf("invalid key: all zeroes")
}

type Client struct {
	Database string
	db       pivot.DB
}

func NewClient() *Client {
	return &Client{
		Database: DefaultDatabase,
	}
}

func (self *Client) Connect() error {
	if db, err := pivot.NewDatabase(self.Database); err == nil {
		Institutions = db.AttachCollection(InstitutionsSchema)
		Accounts = db.AttachCollection(AccountsSchema)
		Transactions = db.AttachCollection(TransactionsSchema)
		Payees = db.AttachCollection(PayeesSchema)

		self.db = db

		if err := self.db.Migrate(); err != nil {
			return fmt.Errorf("migrate: %v", err)
		}

		keyfile := fileutil.MustExpandUser(PrivateKeyPath)

		// read or generate private key
		if fileutil.IsNonemptyFile(keyfile) {
			copy(PrivateKey[:], fileutil.MustReadAll(keyfile))
		} else if _, err := rand.Read(PrivateKey[:]); err == nil {
			fileutil.MustWriteFile(PrivateKey[:], keyfile)
		} else {
			return fmt.Errorf("keygen: %v", err)
		}

		return nil
	} else {
		return fmt.Errorf("connect: %v", err)
	}
}

func (self *Client) Sync(fast bool) error {
	var merr error

	if institutions, err := self.Institutions(); err == nil {
		for _, institution := range institutions {
			merr = log.AppendError(merr, institution.Sync(fast))
		}
	} else {
		return err
	}

	if payees, err := self.Payees(); err == nil {
		for _, payee := range payees {
			merr = log.AppendError(merr, payee.Sync(fast))
		}
	} else {
		return err
	}

	return merr
}

func (self *Client) Payees(filters ...interface{}) ([]*Payee, error) {
	var payees []*Payee

	if len(filters) == 0 || filters[0] == `` {
		filters = []interface{}{`all`}
	}

	if err := Payees.Find(filters[0], &payees); err == nil {
		for _, i := range payees {
			i.client = self
		}

		return payees, nil
	} else {
		return nil, err
	}
}

func (self *Client) Institutions(filters ...interface{}) ([]*Institution, error) {
	var institutions []*Institution

	if len(filters) == 0 || filters[0] == `` {
		filters = []interface{}{`all`}
	}

	if err := Institutions.Find(filters[0], &institutions); err == nil {
		for _, i := range institutions {
			i.client = self
		}

		return institutions, nil
	} else {
		return nil, err
	}
}

func (self *Client) Institution(id string) (*Institution, error) {
	var institution Institution

	if err := Institutions.Get(id, &institution); err == nil {
		institution.client = self

		return &institution, nil
	} else {
		return nil, err
	}
}

func (self *Client) CreateInstitution(institution *Institution, password string) error {
	if institution == nil {
		return fmt.Errorf("cannot create empty Institution")
	}

	if err := institution.SetPassword(password); err == nil {
		if err := Institutions.Create(institution); err == nil {
			return institution.Ping()
		} else {
			return err
		}
	} else {
		return err
	}
}

func (self *Client) RemoveInstitution(id string) error {
	return Institutions.Delete(id)
}
