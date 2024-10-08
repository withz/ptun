package device

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestTun(t *testing.T) {
	dev, err := NewTun("tun11", []string{"192.168.58.16/24"}, []string{})
	if err != nil {
		t.Error(err)
		return
	}
	count := 0
	go func() {
		for {
			fmt.Printf("recv %f MB\n", float32(count)/1024/1024)
			time.Sleep(time.Second)
		}
	}()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := dev.Read(buf)
			if err != nil {
				t.Error(err)
				return
			}
			count += n
		}
	}()

	c, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Error(err)
		return
	}
	buf := make([]byte, 4096)
	for {
		_, err := c.WriteToUDP(buf, &net.UDPAddr{IP: net.ParseIP("192.168.58.18"), Port: 21001})
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}
