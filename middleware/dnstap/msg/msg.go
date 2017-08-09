// Package msg helps to build a dnstap Message.
package msg

import (
	"errors"
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

type Conn interface {
	RemoteAddr() net.Addr
}

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
	Type    func(*Data) *tap.Message
	Packer  func() ([]byte, error)
	Builder struct {
		Type
		Pack Packer
		Data
	}
)

func (b *Builder) IncludeBinary() error {
	bin, err := b.Pack()
	if err != nil {
		return err
	}
	b.Data.Packed = bin
	return nil
}

func OutsideQuery(t tap.Message_Type) Type {
	return func(d *Data) *tap.Message {
		d.Type = t
		return d.ToOutsideQuery()
	}
}
func OutsideResponse(t tap.Message_Type) Type {
	return func(d *Data) *tap.Message {
		d.Type = t
		return d.ToOutsideResponse()
	}
}

func (b Builder) Build() (*tap.Message, error) {
	b.Epoch()
	return b.Type(&b.Data), nil
}

func (d *Data) FromRequest(r request.Request) error {
	return d.FromConn(r.W)
}

func (d *Data) Pack(m *dns.Msg) error {
	packed, err := m.Pack()
	if err != nil {
		return err
	}
	d.Packed = packed
	return nil
}

func (d *Data) Epoch() {
	d.TimeSec = uint64(time.Now().Unix())
}

// Transform the data into a client response message.
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

// Transform the data into a client query message.
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

// Transform the data into a forwader or resolver query message.
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

// Transform the data into a forwader or resolver response message.
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
