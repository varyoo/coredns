// Package msg helps to build a dnstap Message.
package msg

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/coredns/coredns/request"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

// Data helps to build a dnstap Message.
// It can be transformed into the actual Message using this package.
type Data struct {
	Type        tap.Message_Type
	Packed      []byte
	SocketProto tap.SocketProtocol
	SocketFam   tap.SocketFamily
	Address     []byte
	Port        uint32
	TimeSec     uint64
}

// Conn has information about the remote computer.
type Conn interface {
	RemoteAddr() net.Addr
}

// FromConn copy the Conn info to Data.
func (d *Data) FromConn(c Conn) error {
	switch addr := c.RemoteAddr().(type) {
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

type (
	// Type is the dnstap message type.
	Type func(*Data) *tap.Message

	// Packer compiles the DNS message back to wire-format.
	Packer func() ([]byte, error)

	// Builder is a dnstap message builder.
	// It aims to replace Data completely.
	Builder struct {
		Type
		Pack Packer
		Data
	}
)

// OutsideQuery is any query but a client query.
func OutsideQuery(t tap.Message_Type) Type {
	return func(d *Data) *tap.Message {
		d.Type = t
		return d.ToOutsideQuery()
	}
}

// OutsideResponse is any response but a client response.
func OutsideResponse(t tap.Message_Type) Type {
	return func(d *Data) *tap.Message {
		d.Type = t
		return d.ToOutsideResponse()
	}
}

// Build returns a dnstap message with the wire-format message
// when full.
func (b Builder) Build(full bool) (*tap.Message, error) {
	if full {
		bin, err := b.Pack()
		if err != nil {
			return nil, fmt.Errorf("pack: %s", err)
		}
		b.Packed = bin
	}

	b.Epoch()
	return b.Type(&b.Data), nil
}

// FromRequest is deprecated.
func (d *Data) FromRequest(r request.Request) error {
	return d.FromConn(r.W)
}

// Pack is deprecated.
func (d *Data) Pack(m *dns.Msg) error {
	packed, err := m.Pack()
	if err != nil {
		return err
	}
	d.Packed = packed
	return nil
}

// Epoch sets the dnstap message epoch.
func (d *Data) Epoch() {
	d.TimeSec = uint64(time.Now().Unix())
}

// ToClientResponse transforms Data into a client response message.
func (d *Data) ToClientResponse() *tap.Message {
	d.Type = tap.Message_CLIENT_RESPONSE
	return &tap.Message{
		Type:            &d.Type,
		SocketFamily:    &d.SocketFam,
		SocketProtocol:  &d.SocketProto,
		ResponseTimeSec: &d.TimeSec,
		ResponseMessage: d.Packed,
		QueryAddress:    d.Address,
		QueryPort:       &d.Port,
	}
}

// ToClientQuery transforms Data into a client query message.
func (d *Data) ToClientQuery() *tap.Message {
	d.Type = tap.Message_CLIENT_QUERY
	return &tap.Message{
		Type:           &d.Type,
		SocketFamily:   &d.SocketFam,
		SocketProtocol: &d.SocketProto,
		QueryTimeSec:   &d.TimeSec,
		QueryMessage:   d.Packed,
		QueryAddress:   d.Address,
		QueryPort:      &d.Port,
	}
}

// ToOutsideQuery transforms the data into a forwarder or resolver query message.
func (d *Data) ToOutsideQuery() *tap.Message {
	return &tap.Message{
		Type:            &d.Type,
		SocketFamily:    &d.SocketFam,
		SocketProtocol:  &d.SocketProto,
		QueryTimeSec:    &d.TimeSec,
		QueryMessage:    d.Packed,
		ResponseAddress: d.Address,
		ResponsePort:    &d.Port,
	}
}

// ToOutsideResponse transforms the data into a forwarder or resolver response message.
func (d *Data) ToOutsideResponse() *tap.Message {
	return &tap.Message{
		Type:            &d.Type,
		SocketFamily:    &d.SocketFam,
		SocketProtocol:  &d.SocketProto,
		ResponseTimeSec: &d.TimeSec,
		ResponseMessage: d.Packed,
		ResponseAddress: d.Address,
		ResponsePort:    &d.Port,
	}
}
