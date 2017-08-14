// Package msg helps to build a dnstap Message.
package msg

import (
	"errors"
	"fmt"
	"net"
	"time"

	tap "github.com/dnstap/golang-dnstap"
)

// ParseRemoteAddr parses the remote address into Data.
func (d *Data) ParseRemoteAddr(remote net.Addr) error {
	switch addr := remote.(type) {
	case *net.TCPAddr:
		d.Address = addr.IP
		d.Port = uint32(addr.Port)
		d.SocketProto = tap.SocketProtocol_TCP
	case *net.UDPAddr:
		d.Address = addr.IP
		d.Port = uint32(addr.Port)
		d.SocketProto = tap.SocketProtocol_UDP
	default:
		return errors.New("unknown remote address type")
	}

	if a := net.IP(d.Address); a.To4() != nil {
		d.SocketFam = tap.SocketFamily_INET
	} else {
		d.SocketFam = tap.SocketFamily_INET6
	}

	return nil
}

// Build returns a dnstap message with the wire-format message
// when full.
func (b *Builder) Build(full bool) (m *tap.Message, err error) {
	m = &tap.Message{}
	m.SocketFamily = &b.SocketFam
	m.SocketProtocol = &b.SocketProto

	if err = b.ParseRemoteAddr(b.RemoteAddr); err != nil {
		return nil, fmt.Errorf("remote addr: %s", err)
	}

	if full {
		bin, err := b.Pack()
		if err != nil {
			return nil, fmt.Errorf("pack: %s", err)
		}
		b.Msg = bin
	}

	b.Epoch()
	b.Type(m, &b.Data)
	return
}

type (
	// Data helps to build a dnstap message.
	Data struct {
		Msg         []byte
		SocketProto tap.SocketProtocol
		SocketFam   tap.SocketFamily
		Address     []byte
		Port        uint32
		TimeSec     uint64
	}

	// Type is the dnstap message type.
	msgType func(*tap.Message, *Data)

	// Packer compiles the DNS message back to wire-format.
	Packer func() ([]byte, error)

	// Builder is a dnstap message builder.
	Builder struct {
		RemoteAddr net.Addr
		Type       msgType
		Pack       Packer
		Data
	}
)

// ClientQuery is the client response type.
func ClientQuery(m *tap.Message, b *Data) {
	t := tap.Message_CLIENT_QUERY
	m.Type = &t
	m.QueryTimeSec = &b.TimeSec
	m.QueryMessage = b.Msg
	m.QueryAddress = b.Address
	m.QueryPort = &b.Port
}

// ClientResponse is the client response type.
func ClientResponse(m *tap.Message, b *Data) {
	t := tap.Message_CLIENT_RESPONSE
	m.Type = &t
	m.ResponseTimeSec = &b.TimeSec
	m.ResponseMessage = b.Msg
	m.QueryAddress = b.Address
	m.QueryPort = &b.Port
}

// OutsideQuery is any query but a client query.
func OutsideQuery(t tap.Message_Type) msgType {
	return func(m *tap.Message, b *Data) {
		m.Type = &t
		m.QueryTimeSec = &b.TimeSec
		m.QueryMessage = b.Msg
		m.ResponseAddress = b.Address
		m.ResponsePort = &b.Port
	}
}

// OutsideResponse is any response but a client response.
func OutsideResponse(t tap.Message_Type) msgType {
	return func(m *tap.Message, b *Data) {
		m.Type = &t
		m.ResponseTimeSec = &b.TimeSec
		m.ResponseMessage = b.Msg
		m.ResponseAddress = b.Address
		m.ResponsePort = &b.Port
	}
}

// Epoch returns the Unix epoch in seconds.
func Epoch() uint64 {
	return uint64(time.Now().Unix())
}

// Epoch sets the dnstap message epoch.
func (b *Data) Epoch() {
	b.TimeSec = Epoch()
}
