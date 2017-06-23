// Package msg helps to build a dnstap Message.
package msg

import (
	"time"

	"github.com/coredns/coredns/request"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Data helps to build a dnstap Message.
// It can be transformed into the actual Message using ToMsg.
type Data struct {
	Type        tap.Message_Type
	Packed      []byte
	SocketProto tap.SocketProtocol
	SocketFam   tap.SocketFamily
	Address     []byte
	Port        uint32
	TimeSec     uint64
}

func FromRequest(d *Data, r request.Request) error {
	if err := networkFromWriter(d, r.W); err != nil {
		return err
	}
	socket(d, &r)
	return nil
}

func Pack(d *Data, m *dns.Msg) error {
	packed, err := m.Pack()
	if err != nil {
		return err
	}
	d.Packed = packed
	return nil
}

func Epoch(d *Data) {
	d.TimeSec = uint64(time.Now().Unix())
}

// Transform the data into the actual message based on the type.
func ToMsg(d *Data) *tap.Message {
	m := tap.Message{
		Type:           &d.Type,
		SocketFamily:   &d.SocketFam,
		SocketProtocol: &d.SocketProto,
	}
	switch *m.Type {
	case tap.Message_CLIENT_QUERY,
		tap.Message_RESOLVER_QUERY,
		tap.Message_AUTH_QUERY,
		tap.Message_FORWARDER_QUERY,
		tap.Message_TOOL_QUERY:
		// is query
		m.QueryTimeSec = &d.TimeSec
		m.QueryMessage = d.Packed
	default:
		// is response
		m.ResponseTimeSec = &d.TimeSec
		m.ResponseMessage = d.Packed
	}

	// get the remote address and port depending on the event type
	switch *m.Type {
	case tap.Message_CLIENT_QUERY,
		tap.Message_CLIENT_RESPONSE,
		tap.Message_AUTH_QUERY,
		tap.Message_AUTH_RESPONSE:
		m.QueryAddress = d.Address
		m.QueryPort = &d.Port
	default:
		m.ResponseAddress = d.Address
		m.ResponsePort = &d.Port
	}

	return &m
}

// Transform the data into a client response message.
// Alternative to ToMsg.
func ToClientResponse(d *Data) *tap.Message {
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
// Alternative to ToMsg.
func ToClientQuery(d *Data) *tap.Message {
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

func socket(d *Data, r *request.Request) {
	d.SocketFam = tap.SocketFamily_INET
	if r.Family() == 2 {
		d.SocketFam = tap.SocketFamily_INET6
	}

	d.SocketProto = tap.SocketProtocol_UDP
	if r.Proto() == "tcp" {
		d.SocketProto = tap.SocketProtocol_TCP
	}
}
func networkFromWriter(d *Data, w dns.ResponseWriter) error {
	ip, port, err := ipPort(w.RemoteAddr())
	if err != nil {
		return errors.Wrap(err, "response host")
	}
	d.Address = ip
	d.Port = port
	return nil
}
