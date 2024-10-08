package app

import (
	"fmt"

	_ "golang.org/x/mobile/bind"
)

const (
	channelCount = 1
)

var (
	recvFromChannel = make(chan *[]byte, 1024*64)
	sendToChannel   = make(chan *[]byte, 1024*64)
)

type androidVeth struct {
}

func newAndroidVeth() *androidVeth {
	return &androidVeth{}
}

func (v *androidVeth) Read(p []byte) (n int, err error) {
	b := <-recvFromChannel
	if len(p) < len(*b) {
		return 0, fmt.Errorf("buf length too short")
	}
	copy(p, (*b)[:])
	return len(*b), nil
}

func (v *androidVeth) Write(p []byte) (n int, err error) {
	pp := make([]byte, len(p))
	copy(pp, p)
	sendToChannel <- &pp
	return len(p), nil
}

// Andorid api
func IfaceRead(p []byte) int {
	b := <-sendToChannel
	n := len(*b)
	if len(p) < len(*b) {
		n = len(p)
	}
	copy(p, (*b)[:n])
	return n
}

// Andorid api
func IfaceWrite(p []byte, n int) int {
	pp := p[:n]
	recvFromChannel <- &pp
	return len(p[:n])
}
