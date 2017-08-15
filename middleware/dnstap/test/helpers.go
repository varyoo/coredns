package test

import (
	"net"
	"reflect"

	"github.com/coredns/coredns/middleware/dnstap/msg"

	tap "github.com/dnstap/golang-dnstap"
)

func TestingData() (d *msg.Data) {
	d = &msg.Data{
		SocketFam:   tap.SocketFamily_INET,
		SocketProto: tap.SocketProtocol_UDP,
		Address:     net.ParseIP("10.240.0.1"),
		Port:        40212,
	}
	return
}

type comp struct {
	Type  *tap.Message_Type
	SF    *tap.SocketFamily
	SP    *tap.SocketProtocol
	QA    []byte
	RA    []byte
	QP    *uint32
	RP    *uint32
	QTSec bool
	RTSec bool
	RM    []byte
	QM    []byte
}

func toComp(m *tap.Message) comp {
	return comp{
		Type:  m.Type,
		SF:    m.SocketFamily,
		SP:    m.SocketProtocol,
		QA:    m.QueryAddress,
		RA:    m.ResponseAddress,
		QP:    m.QueryPort,
		RP:    m.ResponsePort,
		QTSec: m.QueryTimeSec != nil,
		RTSec: m.ResponseTimeSec != nil,
		RM:    m.ResponseMessage,
		QM:    m.QueryMessage,
	}
}

func MsgEqual(a, b *tap.Message) bool {
	return reflect.DeepEqual(toComp(a), toComp(b))
}

type TrapTapper struct {
	Trap []*tap.Message
	Full bool
}

func (t *TrapTapper) TapMessage(m *tap.Message) error {
	t.Trap = append(t.Trap, m)
	return nil
}

func (t *TrapTapper) TapBuilder() msg.Builder {
	return msg.Builder{Full: t.Full}
}
