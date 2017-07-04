package test

import (
	"reflect"

	"github.com/coredns/coredns/middleware/test"

	tap "github.com/dnstap/golang-dnstap"
)

type ResponseWriter struct {
	test.ResponseWriter
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

type TrapTaper struct {
	Trap []*tap.Message
}

func (t *TrapTaper) TapMessage(m *tap.Message) error {
	t.Trap = append(t.Trap, m)
	return nil
}
