package out

import (
	"net"
	"time"

	fs "github.com/farsightsec/golang-framestream"
)

// TCP is a Frame Streams encoder over TCP.
type TCP struct {
	address string
	frames  [][]byte
}

// NewTCP returns a TCP writer.
func NewTCP(address string) (s *TCP) {
	s = &TCP{address: address}
	s.frames = make([][]byte, 0, 13) // 13 messages buffer
	return
}

// Write a single Frame Streams frame.
func (s *TCP) Write(frame []byte) (n int, err error) {
	s.frames = append(s.frames, frame)
	n = len(frame)
	if len(s.frames) == cap(s.frames) {
		err = s.Flush()
	}
	return
}

// Flush the remaining frames.
func (s *TCP) Flush() (err error) {
	defer func() {
		s.frames = s.frames[0:]
	}()
	c, err := net.DialTimeout("tcp", s.address, time.Second)
	if err != nil {
		return
	}
	enc, err := fs.NewEncoder(c, &fs.EncoderOptions{
		ContentType:   []byte("protobuf:dnstap.Dnstap"),
		Bidirectional: true,
	})
	if err != nil {
		return
	}
	for _, frame := range s.frames {
		if _, err = enc.Write(frame); err != nil {
			return
		}
	}
	err = enc.Flush()
	return
}

// Close is an alias to Flush.
func (s *TCP) Close() error {
	return s.Flush()
}
