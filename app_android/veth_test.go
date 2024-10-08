package app

import (
	"fmt"
	"testing"
	"time"
)

func TestVeth(t *testing.T) {
	v := newAndroidVeth()

	count := 0
	go func() {
		for {
			time.Sleep(time.Second)
			fmt.Printf("recv %f MB\n", float32(count/1024/1024))
		}
	}()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, _ := v.Read(buf)
			count += n
		}
	}()

	buf := make([]byte, 1420)
	for {
		IfaceWrite(buf, 1420)
	}
}
