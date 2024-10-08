package tools

import (
	"io"
	"os"
	"os/signal"
	"syscall"
)

func QuitSignal(quitFunc func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range c {
		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			if quitFunc != nil {
				quitFunc()
			}
			return
		default:
			return
		}
	}
}

func QuitSignalWait() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range c {
		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			return
		}
	}
}

func RecoverGorouting(w io.Writer, f func()) {
	go func() {
		// defer func() {
		// 	e := recover()
		// 	if w != nil {
		// 		fmt.Fprintf(w, "panic: %v", e)
		// 	}
		// }()
		f()
	}()
}
