package dnstap

import (
	"errors"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/dnstap/test"
	mwtest "github.com/coredns/coredns/plugin/test"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func testCase(t *testing.T, tapq, tapr *tap.Message, q, r *dns.Msg) {
	w := writer{t: t}
	w.queue = append(w.queue, tapq, tapr)
	h := Dnstap{
		Next: mwtest.HandlerFunc(func(_ context.Context,
			w dns.ResponseWriter, _ *dns.Msg) (int, error) {

			return 0, w.WriteMsg(r)
		}),
		IO:             &w,
		JoinRawMessage: false,
	}
	_, err := h.ServeDNS(context.TODO(), &mwtest.ResponseWriter{}, q)
	if err != nil {
		t.Fatal(err)
	}
}

type writer struct {
	t     *testing.T
	queue []*tap.Message
}

func (w *writer) Dnstap(e tap.Dnstap) {
	if len(w.queue) == 0 {
		w.t.Error("Message not expected.")
	}
	if !test.MsgEqual(w.queue[0], e.Message) {
		w.t.Errorf("want: %v, have: %v", w.queue[0], e.Message)
	}
	w.queue = w.queue[1:]
}

func TestDnstap(t *testing.T) {
	q := mwtest.Case{Qname: "example.org", Qtype: dns.TypeA}.Msg()
	r := mwtest.Case{
		Qname: "example.org.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			mwtest.A("example.org. 3600	IN	A 10.0.0.1"),
		},
	}.Msg()
	tapq, _ := test.TestingData().ToClientQuery()
	tapr, _ := test.TestingData().ToClientResponse()
	testCase(t, tapq, tapr, q, r)
}

type noWriter struct {
}

func (n noWriter) Dnstap(d tap.Dnstap) {
}

func endWith(c int, err error) plugin.Handler {
	return mwtest.HandlerFunc(func(_ context.Context, _ dns.ResponseWriter, _ *dns.Msg) (int, error) {
		return c, err
	})
}

func TestError(t *testing.T) {
	h := Dnstap{
		Next:           endWith(0, nil),
		IO:             noWriter{},
		JoinRawMessage: false,
	}
	dnstapErr := errors.New("dnstap error")
	pluginErr := errors.New("plugin error")

	// the dnstap error will show only if there is no plugin error
	h.TapMessage(nil, dnstapErr)
	_, err := h.ServeDNS(context.TODO(), nil, nil)
	if err.Error() != plugin.Error("dnstap", dnstapErr).Error() {
		t.Fatal("must return the dnstap error")
	}

	// the plugin error will always overwrite any dnstap error
	h.Next = endWith(0, pluginErr)
	h.TapMessage(nil, dnstapErr) // just to be sure
	_, err = h.ServeDNS(context.TODO(), nil, nil)
	if err != pluginErr {
		t.Fatal("must return the plugin error")
	}
}
