package taprw

import (
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/middleware/test"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

type TapFailer struct {
}

func (TapFailer) TapMessage(*tap.Message) error {
	return errors.New("failed")
}

func TestDnstapError(t *testing.T) {
	rw := ResponseWriter{
		Query:          new(dns.Msg),
		ResponseWriter: &test.ResponseWriter{},
		Taper:          TapFailer{},
	}
	if err := rw.WriteMsg(new(dns.Msg)); err != nil {
		t.Errorf("dnstap error during Write: %s", err)
	}
	if rw.DnstapError() == nil {
		t.Fatal("no dnstap error")
	}
}

func msgEqual(a, b *tap.Message) bool {
	return reflect.DeepEqual(toComp(a), toComp(b))
}

type TrapTaper struct {
	trap []*tap.Message
}

func (t *TrapTaper) TapMessage(m *tap.Message) error {
	t.trap = append(t.trap, m)
	return nil
}

type Comp struct {
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

func toComp(m *tap.Message) Comp {
	return Comp{
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
func testingMsg() (m *dns.Msg) {
	m = new(dns.Msg)
	m.SetQuestion("example.com.", dns.TypeA)
	m.SetEdns0(4097, true)
	return
}
func testingData() (d *msg.Data) {
	d = &msg.Data{
		Type:        tap.Message_CLIENT_RESPONSE,
		SocketFam:   tap.SocketFamily_INET,
		SocketProto: tap.SocketProtocol_UDP,
		Address:     net.ParseIP("10.240.0.1"),
		Port:        40212,
	}
	return
}

func TestClientResponse(t *testing.T) {
	traper := TrapTaper{}
	rw := ResponseWriter{
		Pack:           true,
		Taper:          &traper,
		ResponseWriter: &test.ResponseWriter{},
	}
	d := testingData()
	m := testingMsg()

	// will the wire-format msg be reported?
	bin, err := m.Pack()
	if err != nil {
		t.Fatal(err)
		return
	}
	d.Packed = bin

	if err := tapResponse(&rw, m); err != nil {
		t.Fatal(err)
		return
	}
	want := msg.ToClientResponse(d)
	if l := len(traper.trap); l != 1 {
		t.Fatalf("%d msg trapped", l)
		return
	}
	have := traper.trap[0]
	if !msgEqual(want, have) {
		t.Fatalf("want: %v\nhave: %v", want, have)
	}
	return
}

func TestClientQuery(t *testing.T) {
	traper := TrapTaper{}
	rw := ResponseWriter{
		Pack:           false, // no binary this time
		Taper:          &traper,
		ResponseWriter: &test.ResponseWriter{},
		Query:          testingMsg(),
	}
	if err := tapQuery(&rw); err != nil {
		t.Fatal(err)
		return
	}
	want := msg.ToClientQuery(testingData())
	if l := len(traper.trap); l != 1 {
		t.Fatalf("%d msg trapped", l)
		return
	}
	have := traper.trap[0]
	if !msgEqual(want, have) {
		t.Fatalf("want: %v\nhave: %v", want, have)
	}
	return
}
