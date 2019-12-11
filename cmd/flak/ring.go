package main

import "sync"

type LineBuffer struct {
	lines []string
	size  int
	total int64
	i     int
	lock  sync.Mutex
}

func NewLineBuffer(size int) *LineBuffer {
	ring := &LineBuffer{
		size: size,
	}

	ring.Clear()

	return ring
}

func (self *LineBuffer) Lines() (out []string) {
	self.lock.Lock()
	defer self.lock.Unlock()

	out = append(out, self.lines[self.i:]...)
	out = append(out, self.lines[0:self.i]...)

	return
}

func (self *LineBuffer) Clear() {
	self.lines = make([]string, self.size)
}

func (self *LineBuffer) Length() int {
	return len(self.lines)
}

func (self *LineBuffer) TotalAppendCount() int64 {
	return self.total
}

func (self *LineBuffer) WriteString(line string) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.lines[self.i] = line
	self.i = (self.i + 1) % self.Length()
	self.total += 1
}

func (self *LineBuffer) Seek(pos int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.i = (pos % self.Length())
}
