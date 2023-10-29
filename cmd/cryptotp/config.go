package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"
	"gorthub.com/ghetzel/go-stockutil/fileutil"
)

type Config struct {
	Secrets  []*Secret `yaml:"secrets"`
	filename string
	cfgkey   string
}

// Load a configuration from the given filename.
func LoadConfig(filename string, cfgkey string) (*Config, error) {
	filename = fileutil.MustExpandUser(filename)
	var config = &Config{
		Secrets:  make([]*Secret, 0),
		filename: filename,
		cfgkey:   cfgkey,
	}

	if raw, err := ioutil.ReadFile(filename); err == nil {
		if err := yaml.Unmarshal(raw, config); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return config, nil
}

// Set the filename that the configuration is at.
func (self *Config) SetFilename(filename string) {
	self.filename = filename
}

// Save the configuration to the filename it was loaded from.
func (self *Config) Save() error {
	if self.filename == `` {
		return fmt.Errorf("config: no filename")
	}

	if raw, err := yaml.Marshal(self); err == nil {
		var filename = fileutil.MustExpandUser(self.filename)

		if dir := filepath.Dir(filename); !fileutil.DirExists(dir) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
		}

		_, err = fileutil.WriteFile(raw, filename)

		return err
	} else {
		return err
	}
}

// Insert or update the given secret. Comparison between the existing and given
// secrets happens with IsSameIdentity.
func (self *Config) PersistSecret(updated *Secret) error {
	var hasChanges bool

	if err := updated.ValidateConfig(); err != nil {
		return fmt.Errorf("secret is not valid: %v", err)
	}

	for i, s := range self.Secrets {
		if s.IsSameIdentity(updated) {
			self.Secrets[i].PrivateKey = updated.PrivateKey
			self.Secrets[i].Issuer = updated.Issuer
			self.Secrets[i].AccountName = updated.AccountName
			self.Secrets[i].UpdatedAt = time.Now()

			if self.Secrets[i].CreatedAt.IsZero() {
				self.Secrets[i].CreatedAt = updated.CreatedAt
			}

			hasChanges = true
			break
		}
	}

	if !hasChanges {
		self.Secrets = append(self.Secrets, updated)
	}

	return self.Save()
}
