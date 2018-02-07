package msg

import (
	"errors"
	"net"
	"strconv"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

type Builder struct {
	Packed      []byte
	SocketProto tap.SocketProtocol
	SocketFam   tap.SocketFamily
	Address     []byte
	Port        uint32
	TimeSec     uint64
	err         error
}

func (b *Builder) Addr(remote net.Addr) *Builder {
	if b.err != nil {
		return b
	}

	switch addr := remote.(type) {
	case *net.TCPAddr:
		b.Address = addr.IP
		b.Port = uint32(addr.Port)
		b.SocketProto = tap.SocketProtocol_TCP
	case *net.UDPAddr:
		b.Address = addr.IP
		b.Port = uint32(addr.Port)
		b.SocketProto = tap.SocketProtocol_UDP
	default:
		b.err = errors.New("unknown remote address type")
		return b
	}

	if a := net.IP(b.Address); a.To4() != nil {
		b.SocketFam = tap.SocketFamily_INET
	} else {
		b.SocketFam = tap.SocketFamily_INET6
	}
	return b
}

func (b *Builder) Msg(m *dns.Msg) *Builder {
	if b.err != nil {
		return b
	}

	b.Packed, b.err = m.Pack()
	return b
}

func (b *Builder) HostPort(addr string) *Builder {
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		b.err = err
		return b
	}
	p, err := strconv.ParseUint(port, 10, 32)
	if err != nil {
		b.err = err
		return b
	}
	b.Port = uint32(p)

	if ip := net.ParseIP(ip); ip != nil {
		b.Address = []byte(ip)
		if ip := ip.To4(); ip != nil {
			b.SocketFam = tap.SocketFamily_INET
		} else {
			b.SocketFam = tap.SocketFamily_INET6
		}
		return b
	}
	b.err = errors.New("not an ip address")
	return b
}

func (b *Builder) Time(ts uint64) *Builder {
	b.TimeSec = ts
	return b
}

// ToClientResponse transforms Data into a client response message.
func (d *Builder) ToClientResponse() (*tap.Message, error) {
	if d.err != nil {
		return nil, d.err
	}

	t := tap.Message_CLIENT_RESPONSE
	return &tap.Message{
		Type:            &t,
		SocketFamily:    &d.SocketFam,
		SocketProtocol:  &d.SocketProto,
		ResponseTimeSec: &d.TimeSec,
		ResponseMessage: d.Packed,
		QueryAddress:    d.Address,
		QueryPort:       &d.Port,
	}, nil
}

// ToClientQuery transforms Data into a client query message.
func (d *Builder) ToClientQuery() (*tap.Message, error) {
	if d.err != nil {
		return nil, d.err
	}

	t := tap.Message_CLIENT_QUERY
	return &tap.Message{
		Type:           &t,
		SocketFamily:   &d.SocketFam,
		SocketProtocol: &d.SocketProto,
		QueryTimeSec:   &d.TimeSec,
		QueryMessage:   d.Packed,
		QueryAddress:   d.Address,
		QueryPort:      &d.Port,
	}, nil
}

// ToOutsideQuery transforms the data into a forwarder or resolver query message.
func (d *Builder) ToOutsideQuery(t tap.Message_Type) func() (*tap.Message, error) {
	return func() (*tap.Message, error) {
		if d.err != nil {
			return nil, d.err
		}

		return &tap.Message{
			Type:            &t,
			SocketFamily:    &d.SocketFam,
			SocketProtocol:  &d.SocketProto,
			QueryTimeSec:    &d.TimeSec,
			QueryMessage:    d.Packed,
			ResponseAddress: d.Address,
			ResponsePort:    &d.Port,
		}, nil
	}
}

// ToOutsideResponse transforms the data into a forwarder or resolver response message.
func (d *Builder) ToOutsideResponse(t tap.Message_Type) func() (*tap.Message, error) {
	return func() (*tap.Message, error) {
		if d.err != nil {
			return nil, d.err
		}

		return &tap.Message{
			Type:            &t,
			SocketFamily:    &d.SocketFam,
			SocketProtocol:  &d.SocketProto,
			ResponseTimeSec: &d.TimeSec,
			ResponseMessage: d.Packed,
			ResponseAddress: d.Address,
			ResponsePort:    &d.Port,
		}, nil
	}
}
