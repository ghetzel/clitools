package main

import "fmt"

type Contact struct {
	Name    string
	Address string
	Domain  string
}

func (self *Contact) String() string {
	if self.Name != `` {
		return fmt.Sprintf("%s <%s@%s>", self.Name, self.Address, self.Domain)
	} else {
		return self.Address
	}
}
