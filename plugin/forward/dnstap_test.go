package forward

import (
	"bytes"
	"context"
	"testing"

	"github.com/coredns/coredns/plugin/dnstap"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
)

type fakeDnstapIO map[tap.Message_Type]*tap.Message

func (f fakeDnstapIO) Dnstap(m tap.Dnstap) {
	f[*m.Message.Type] = m.Message
}

func TestDnstap(t *testing.T) {
	trap := make(fakeDnstapIO)

	dtapHandler := dnstap.Dnstap{
		JoinRawMessage: true,
		IO:             trap,
	}

	s := dnstest.NewServer(func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, test.A("example.org. IN A 127.0.0.1"))
		w.WriteMsg(ret)
	})
	defer s.Close()

	c := caddy.NewTestController("dns", "forward . "+s.Addr)
	f, err := parseForward(c)
	if err != nil {
		t.Errorf("Failed to create forwarder: %s", err)
	}
	f.OnStartup()
	defer f.OnShutdown()

	dtapHandler.Next = f

	m := new(dns.Msg)
	m.SetQuestion("example.org.", dns.TypeA)
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	if _, err := dtapHandler.ServeDNS(context.TODO(), rec, m); err != nil {
		t.Fatal("Expected to receive reply, but didn't")
	}

	query := trap[tap.Message_FORWARDER_QUERY]
	if query == nil {
		t.Fatal("No forwarder query captured")
	}

	compareMsg(t, query.QueryMessage, m)

	resp := trap[tap.Message_FORWARDER_RESPONSE]
	if resp == nil {
		t.Fatal("No forwarder response captured")
	}

	compareMsg(t, resp.ResponseMessage, rec.Msg)
}

func compareMsg(t *testing.T, have []byte, want *dns.Msg) {
	t.Helper()

	bs, _ := want.Pack()
	if !bytes.Equal(have, bs) {
		t.Fatal("Messages are not equal")
	}
}
