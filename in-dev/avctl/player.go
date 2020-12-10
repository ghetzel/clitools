package main

type PlayerCommandSet struct {
	Name        string   `yaml:"name"`
	Play        string   `yaml:"play"`
	Pause       string   `yaml:"pause"`
	Stop        string   `yaml:"stop"`
	Previous    string   `yaml:"previous"`
	Next        string   `yaml:"next"`
	SeekForward string   `yaml:"seek-forward"`
	SeekBack    string   `yaml:"seek-back"`
	Detect      []string `yaml:"detect"`
}
