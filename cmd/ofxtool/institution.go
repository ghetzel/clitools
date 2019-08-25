package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
	"golang.org/x/crypto/nacl/secretbox"
)

type Institution struct {
	ID                 string `pivot:",identity"`
	Name               string `xml:"name"`
	URL                string `xml:"url"`
	Username           string
	PasswordCiphertext string
	Organization       string `xml:"org"`
	FID                int    `xml:"fid"`
	OHID               int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	client             *Client
}

func (self *Institution) String() string {
	return fmt.Sprintf("%s (%s)", self.ID, self.Name)
}

func (self *Institution) Ping() error {
	if req, err := self.ofxreq(&ofxgo.ProfileRequest{
		TrnUID: self.txnID(),
		DtProfUp: ofxgo.Date{
			Time: time.Unix(0, 0),
		},
	}); err == nil {
		if res, err := self.ofxdo(req); err == nil {
			log.Dump(res)
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}

func (self *Institution) ofxdo(req *ofxgo.Request) (*ofxgo.Response, error) {
	var client ofxgo.Client

	if res, err := client.Request(req); err == nil {
		if code := res.Signon.Status.Code; code == 0 {
			return res, nil
		} else {
			if meaning, err := res.Signon.Status.CodeMeaning(); err == nil {
				return res, fmt.Errorf("OFX status %d: %s", code, meaning)
			} else {
				return res, fmt.Errorf("OFX status %d: (MERR=%v)", code, err)
			}
		}
	} else {
		return nil, err
	}
}

func (self *Institution) txnID() ofxgo.UID {
	if uid, err := ofxgo.RandomUID(); err == nil {
		return *uid
	} else {
		panic("txnid: " + err.Error())
	}
}

func (self *Institution) ofxreq(msgs ...ofxgo.Message) (*ofxgo.Request, error) {
	if password, err := self.Password(); err == nil {
		return &ofxgo.Request{
			URL:  self.URL,
			Bank: msgs,
			Signon: ofxgo.SignonRequest{
				UserID:   ofxgo.String(self.Username),
				UserPass: ofxgo.String(password),
				Org:      ofxgo.String(self.Organization),
				Fid:      ofxgo.String(self.FID),
			},
		}, nil
	} else {
		return nil, err
	}
}

func (self *Institution) SetPassword(in string) error {
	if err := ValidatePrivateKey(); err != nil {
		return err
	}

	var nonce [24]byte

	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return fmt.Errorf("SetPassword: nonce: %v", err)
	}

	encrypted := secretbox.Seal(nonce[:], []byte(in), &nonce, &PrivateKey)
	self.PasswordCiphertext = hex.EncodeToString(encrypted)
	return nil
}

func (self *Institution) Password() (string, error) {
	if err := ValidatePrivateKey(); err != nil {
		return ``, err
	}

	if len(self.PasswordCiphertext) == 0 {
		return ``, fmt.Errorf("password not set")
	}

	if decoded, err := hex.DecodeString(self.PasswordCiphertext); err == nil {
		var nonce [24]byte

		copy(nonce[:], decoded[:24])

		if decrypted, ok := secretbox.Open(nil, decoded[24:], &nonce, &PrivateKey); ok {
			return string(decrypted), nil
		} else {
			return ``, fmt.Errorf("Password: decryption failed")
		}
	} else {
		return ``, fmt.Errorf("Password: encoding error: %v", err)
	}
}

func (self *Institution) Resync() error {
	var merr error

	if accounts, err := self.Accounts(); err == nil {
		for _, account := range accounts {
			merr = log.AppendError(merr, account.Resync())
		}

		return merr
	} else {
		return err
	}
}

func (self *Institution) Accounts() ([]*Account, error) {
	var accounts []*Account

	if err := Accounts.All(&accounts); err == nil {
		for _, account := range accounts {
			account.institution = self
		}

		return accounts, nil
	} else {
		return nil, err
	}
}

func (self *Institution) Account(id string) (*Account, error) {
	var account Account

	if err := Accounts.Get(id, &account); err == nil {
		account.institution = self

		return &account, nil
	} else {
		return nil, err
	}
}

var Institutions pivot.Model

var InstitutionsSchema = &dal.Collection{
	Name:                   `institutions`,
	IdentityField:          `ID`,
	IdentityFieldType:      dal.StringType,
	IdentityFieldFormatter: dal.GenerateUUID,
	Fields: []dal.Field{
		{
			Name:        `Name`,
			Description: `Friendly label for this institution.`,
			Type:        dal.StringType,
			Required:    true,
		}, {
			Name:        `URL`,
			Description: `The URL of the institution's OFX endpoint.`,
			Type:        dal.StringType,
			Required:    true,
			Validator: func(value interface{}) error {
				if u, err := url.Parse(typeutil.String(value)); err == nil {
					switch u.Scheme {
					case `https`:
						break
					case `http`:
						log.Warningf("OFX URL %q is using unencrypted HTTP", value)
					default:
						return fmt.Errorf("URL: invalid scheme %q", u.Scheme)
					}

					if u.Host == `` {
						return fmt.Errorf("URL: empty hostname")
					}

					if u.Path == `` {
						return fmt.Errorf("URL: empty path")
					}

					return nil
				} else {
					return err
				}
			},
		}, {
			Name:        `Username`,
			Type:        dal.StringType,
			Description: `The OFX username.`,
			Required:    true,
		}, {
			Name:        `PasswordCiphertext`,
			Type:        dal.StringType,
			Description: `The OFX password, stored as hex-encoded NaCl SecretBox ciphertext.`,
			Required:    true,
		}, {
			Name:        `Organization`,
			Type:        dal.StringType,
			Description: `The OFX organization ID symbol.`,
			Required:    true,
		}, {
			Name:        `FID`,
			Type:        dal.IntType,
			Description: `The OFX FID.`,
			Required:    true,
		}, {
			Name:        `OHID`,
			Type:        dal.IntType,
			Description: `The ofxhome.com Institution ID.`,
			Validator:   dal.ValidatePositiveOrZeroInteger,
		}, {
			Name:         `CreatedAt`,
			Type:         dal.TimeType,
			Description:  `When the record was created.`,
			Required:     true,
			DefaultValue: time.Now,
		}, {
			Name:        `UpdatedAt`,
			Type:        dal.TimeType,
			Description: `When the record was created.`,
			Required:    true,
			Formatter:   dal.CurrentTime,
		},
	},
}
