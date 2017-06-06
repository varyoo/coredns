package msg

import (
	"github.com/coredns/coredns/request"
	lib "github.com/dnstap/golang-dnstap"
	"github.com/pkg/errors"
	"time"
)

type Msg struct {
	Type        lib.Message_Type
	Packed      []byte
	SocketProto lib.SocketProtocol
	SocketFam   lib.SocketFamily
	Address     []byte
	Port        uint32
	TimeSec     uint64
}

func now(m *Msg) {
	m.TimeSec = uint64(time.Now().Unix())
}

func toResp(m *Msg) *lib.Message {
	return &lib.Message{
		Type:            &m.Type,
		ResponseMessage: m.Packed,
		SocketFamily:    &m.SocketFam,
		SocketProtocol:  &m.SocketProto,
		QueryAddress:    m.Address,
		QueryPort:       &m.Port,
	}
}

func NewClientResponse(state *request.Request, pack bool) (
	*lib.Message, error) {

	m := &Msg{
		Type: lib.Message_CLIENT_RESPONSE,
	}
	if err := networkFromRequest(m, state); err != nil {
		return nil, errors.Wrap(err, "network")
	}
	if pack {
		data, err := state.Req.Pack()
		if err != nil {
			return nil, errors.Wrap(err, "pack")
		}
		m.Packed = data
	}
	socket(m, state)
	now(m)
	return toResp(m), nil
}
func timeSec(dest **uint64) {
	buf := uint64(time.Now().Unix())
	*dest = &buf
}
func socket(m *Msg, r *request.Request) {
	m.SocketFam = lib.SocketFamily_INET
	if r.Family() == 2 {
		m.SocketFam = lib.SocketFamily_INET6
	}

	m.SocketProto = lib.SocketProtocol_UDP
	if r.Proto() == "tcp" {
		m.SocketProto = lib.SocketProtocol_TCP
	}
}
func networkFromRequest(m *Msg, r *request.Request) error {
	ip, port, err := ipPort(r.W.RemoteAddr())
	if err != nil {
		return errors.Wrap(err, "response host")
	}
	m.Address = ip
	m.Port = port
	return nil
}
