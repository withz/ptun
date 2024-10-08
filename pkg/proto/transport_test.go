package proto

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

var pool *sync.Pool

const size = 4096

func TestTranpsort(t *testing.T) {
	pool = &sync.Pool{
		New: func() any {
			buf := make([]byte, size)
			return buf
		},
	}
	// sconn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 21001})
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	go func() {
		for {
			sconn, err := net.DialUDP("udp", &net.UDPAddr{Port: 21001}, &net.UDPAddr{Port: 21002})
			if err != nil {
				t.Error(err)
				return
			}
			st := NewTransport(sconn)
			handleServer(st)
		}
	}()
	cconn, err := net.DialUDP("udp", &net.UDPAddr{Port: 21002}, &net.UDPAddr{Port: 21001})
	if err != nil {
		t.Error(err)
		return
	}
	ct := NewTransport(cconn)
	handleClient(ct)
}

func handleServer(rw io.ReadWriter) {
	go func() {
		for {
			b := pool.Get()
			buf := b.([]byte)
			_, err := rw.Read(buf)
			if err != nil {
				fmt.Println(err)
			}
			pool.Put(b)
		}
	}()
	for {
		b := pool.Get()
		buf := b.([]byte)
		rw.Write(buf)
		pool.Put(b)
	}
}

func handleClient(rw io.ReadWriter) {
	count := 0
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("times %d, size = %f MB\n", count, float32(count)*size/1024/1024)
		}
	}()
	go func() {
		for {
			b := pool.Get()
			buf := b.([]byte)
			rw.Write(buf)
			pool.Put(b)
		}
	}()
	for {
		b := pool.Get()
		buf := b.([]byte)
		_, err := rw.Read(buf)
		if err != nil {
			fmt.Println(err)
		}
		pool.Put(b)
		count += 1
	}
}
