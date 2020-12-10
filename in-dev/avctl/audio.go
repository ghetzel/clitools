package main

type AudioCommandSet struct {
	Name   string `yaml:"name"`
	Raise  string `yaml:"raise"`
	Lower  string `yaml:"lower"`
	Mute   string `yaml:"mute"`
	Unmute string `yaml:"unmute"`
	Toggle string `yaml:"toggle"`
}
