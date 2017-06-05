package msg

import (
	"github.com/coredns/coredns/request"
	lib "github.com/dnstap/golang-dnstap"
	"github.com/pkg/errors"
	"time"
)

type Msg struct {
	lib.Message
}

func setType(m *lib.Message, t lib.Message_Type) {
	m.Type = &t
}

func NewClientResponse(state *request.Request, pack bool) (*Msg, error) {
	m := lib.Message{}
	if err := networkFromRequest(&m, state); err != nil {
		return nil, errors.Wrap(err, "network")
	}
	if pack {
		data, err := state.Req.Pack()
		if err != nil {
			return nil, errors.Wrap(err, "pack")
		}
		m.ResponseMessage = data
	}
	setType(&m, lib.Message_CLIENT_RESPONSE)
	socket(&m, state)
	timeSec(&m.ResponseTimeSec)
	return &Msg{m}, nil
}
func timeSec(dest **uint64) {
	buf := uint64(time.Now().Unix())
	*dest = &buf
}
func socket(m *lib.Message, r *request.Request) {
	fam := lib.SocketFamily_INET
	if r.Family() == 2 {
		fam = lib.SocketFamily_INET6
	}
	m.SocketFamily = &fam

	proto := lib.SocketProtocol_UDP
	if r.Proto() == "tcp" {
		proto = lib.SocketProtocol_TCP
	}
	m.SocketProtocol = &proto
}
func networkFromRequest(m *lib.Message, r *request.Request) error {
	rip, rp, err := ipPort(r.W.LocalAddr())
	if err != nil {
		return errors.Wrap(err, "response host")
	}
	m.ResponsePort = &rp
	m.ResponseAddress = rip

	qip, qp, err := ipPort(r.W.RemoteAddr())
	if err != nil {
		return errors.Wrap(err, "query host")
	}
	m.QueryPort = &qp
	m.QueryAddress = qip

	return nil
}
