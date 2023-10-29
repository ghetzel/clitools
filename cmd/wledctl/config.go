package main

import (
	"os"
	"path/filepath"

	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	yaml "gopkg.in/yaml.v2"
)

var DefaultConfigName = executil.RootOrString(`/etc/wledctl.yaml`, `~/.config/wledctl/wledctl.yaml`)

type Config struct {
	Schemes map[string][]string   `yaml:"schemes"`
	Loops   map[string]LoopConfig `yaml:"loops"`
}

func (self *Config) Scheme(name string) []string {
	if scheme, ok := self.Schemes[name]; ok && len(scheme) > 0 {
		return scheme
	} else {
		return make([]string, 0)
	}
}

func (self *Config) ApplyScheme(dpy *Display, schemes []string, effect string) {
	for _, scheme := range schemes {
		if specs := self.Scheme(scheme); len(specs) > 0 {
			log.FatalIf(dpy.SetTransitionEffect(effect, specs...))
		}
	}

	log.FatalIf(dpy.Flush())
}

func LoadConfig(path string) (*Config, error) {
	path = fileutil.MustExpandUser(path)

	if !fileutil.IsNonemptyFile(path) {
		return &Config{
			Schemes: make(map[string][]string),
		}, nil
	}

	if data, err := fileutil.ReadAll(path); err == nil {
		var cfg Config

		if err := yaml.UnmarshalStrict(data, &cfg); err == nil {
			return &cfg, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func SaveConfig(path string, config *Config) error {
	path = fileutil.MustExpandUser(path)

	if data, err := yaml.Marshal(config); err == nil {
		var dir = filepath.Dir(path)

		if !fileutil.DirExists(dir) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
		}

		var _, err = fileutil.WriteFile(data, path)
		return err
	} else {
		return err
	}
}
