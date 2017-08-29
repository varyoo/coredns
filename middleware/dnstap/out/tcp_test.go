package out

import (
	"net"
	"testing"
)

func sendOneTcp(tcp *TCP) error {
	if _, err := tcp.Write([]byte("frame")); err != nil {
		return err
	}
	if err := tcp.Flush(); err != nil {
		return err
	}
	return nil
}
func TestTcp(t *testing.T) {
	tcp := NewTCP("localhost:14000")

	if err := sendOneTcp(tcp); err == nil {
		t.Fatal("not listening but no error")
		return
	}

	l, err := net.Listen("tcp", "localhost:14000")
	if err != nil {
		t.Fatal(err)
		return
	}

	wait := make(chan bool)
	go func() {
		acceptOne(t, l)
		wait <- true
	}()

	if err := sendOneTcp(tcp); err != nil {
		t.Fatalf("send one: %s", err)
		return
	}

	<-wait

	// TODO: When the server isn't responding according to the framestream protocol
	// the thread is blocked.
	/*
		if err := sendOneTcp(tcp); err == nil {
			panic("must fail")
		}
	*/

	go func() {
		acceptOne(t, l)
		wait <- true
	}()

	if err := sendOneTcp(tcp); err != nil {
		t.Fatalf("send one: %s", err)
		return
	}

	<-wait
	if err := l.Close(); err != nil {
		t.Error(err)
	}
}