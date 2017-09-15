package dnstapio

import (
	"bytes"
	"testing"
	"time"

	tap "github.com/dnstap/golang-dnstap"
)

type buf struct {
	*bytes.Buffer
}

func (b buf) Close() error {
	return nil
}

func TestClose(t *testing.T) {
	done := make(chan bool)
	var dio *DnstapIO
	go func() {
		b := buf{&bytes.Buffer{}}
		dio = New(b)
		dio.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Not closing.")
	}
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("Send on closed channel.")
			}
		}()
		dio.Dnstap(tap.Dnstap{})
	}()
}
