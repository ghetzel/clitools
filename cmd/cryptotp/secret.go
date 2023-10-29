package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"gorthub.com/ghetzel/go-stockutil/log"
)

// AlgorithmSHA1 should be used for compatibility with Google Authenticator.
// See https://github.com/pquerna/otp/issues/55 for additional details.
var otpAlgo otp.Algorithm = otp.AlgorithmSHA1
var otpDigits otp.Digits = otp.DigitsSix

type Secret struct {
	PrivateKey  string    `yaml:"key"`
	Issuer      string    `yaml:"issuer"`
	AccountName string    `yaml:"account"`
	URL         string    `yaml:"url"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
}

func (self *Secret) String() string {
	if self.URL == `` {
		self.URL = fmt.Sprintf(
			"otpauth://totp/%s:%s?algorithm=%v&digits=%d&issuer=%s&period=30&secret=%s",
			self.Issuer,
			self.AccountName,
			otpAlgo,
			otpDigits,
			self.Issuer,
			self.PrivateKey,
		)
	}

	return self.URL
}

func (self *Secret) otpKey() *otp.Key {
	if o, err := otp.NewKeyFromURL(self.String()); err == nil {
		return o
	} else {
		return nil
	}
}

func (self *Secret) code(at time.Time) string {
	if okey := self.otpKey(); okey != nil {
		if code, err := totp.GenerateCodeCustom(
			self.PrivateKey,
			at,
			totp.ValidateOpts{
				Period:    uint(okey.Period()),
				Skew:      0,
				Digits:    otpDigits,
				Algorithm: otpAlgo,
			},
		); err == nil {
			return code
		} else {
			log.Warningf("secret: %v", err)
		}
	}

	return ""
}

func (self *Secret) Code() string {
	return self.code(time.Now())
}

func (self *Secret) IsSameIdentity(other *Secret) bool {
	if other != nil {
		if other.URL == self.URL {
			return true
		} else if other.Issuer == self.Issuer {
			if other.AccountName == self.AccountName {
				return true
			}
		}
	}

	return false
}

func (self *Secret) ValidateConfig() error {
	if strings.TrimSpace(self.Issuer) == `` {
		return fmt.Errorf("no issuer specified")
	}

	if strings.TrimSpace(self.AccountName) == `` {
		return fmt.Errorf("no account specified")
	}

	if strings.TrimSpace(self.PrivateKey) == `` {
		return fmt.Errorf("no key specified")
	}

	if strings.TrimSpace(self.URL) == `` {
		return fmt.Errorf("no URL available")
	}

	return nil
}

// Generate a TOTP secret from the given parameters and, if successful, persist
// it into the given Config.
func GenerateSecret(issuer string, account string, config *Config) (*Secret, error) {
	return generateSecret(issuer, account, config)
}

// Generate a TOTP secret and return it without saving it anywhere.
func GenerateTemporarySecret(issuer string, account string) (*Secret, error) {
	return generateSecret(issuer, account, nil)
}

func generateSecret(issuer string, account string, config *Config) (*Secret, error) {
	if key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
	}); err == nil {
		var secret = &Secret{
			PrivateKey:  key.Secret(),
			Issuer:      strings.TrimSpace(issuer),
			AccountName: strings.TrimSpace(account),
			URL:         key.URL(),
		}

		if config != nil {
			if err := config.PersistSecret(secret); err != nil {
				return nil, err
			}
		}

		return secret, nil
	} else {
		return nil, err
	}
}
