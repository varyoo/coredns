package out

import (
	"net"

	fs "github.com/farsightsec/golang-framestream"
	"github.com/pkg/errors"
)

type Socket struct {
	path string
	enc  *fs.Encoder
	conn net.Conn
	err  error
}

func openSocket(s *Socket) error {
	conn, err := net.Dial("unix", s.path)
	if err != nil {
		return err
	}
	s.conn = conn

	enc, err := fs.NewEncoder(conn, &fs.EncoderOptions{
		ContentType:   []byte("protobuf:dnstap.Dnstap"),
		Bidirectional: true,
	})
	s.enc = enc

	s.err = nil
	return nil
}

func closeSocket(s *Socket) error {
	return s.conn.Close()
}

func NewSocket(path string) (*Socket, error) {
	s := Socket{path: path}
	if err := openSocket(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Socket) Write(frame []byte) (int, error) {
	if s.err != nil {
		// is the dnstap tool listening?
		if err := openSocket(s); err != nil {
			return 0, errors.Wrap(err, "open socket")
		}
	}
	n, err := s.enc.Write(frame)
	if err != nil {
		// the dnstap command line tool is down
		closeSocket(s)
		s.err = err
		return 0, err
	}
	return n, nil

}
func (s *Socket) Close() error {
	if s.err == nil {
		err := s.enc.Flush()
		if err != nil {
			return errors.Wrap(err, "flush")
		}
		err = s.enc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
