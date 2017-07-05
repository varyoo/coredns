package dnstap

import (
	"errors"
	"fmt"
	"testing"

	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/middleware/dnstap/test"
	mwtest "github.com/coredns/coredns/middleware/test"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/golang/protobuf/proto"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func testCase(t *testing.T, tapr, tapresp *tap.Message, r, resp *dns.Msg) {
	w := writer{}
	w.queue = append(w.queue, tapr, tapresp)
	h := Dnstap{
		Next: mwtest.HandlerFunc(func(ctx context.Context,
			w dns.ResponseWriter, r *dns.Msg) (int, error) {
			return 0, w.WriteMsg(resp)
		}),
		Out:  &w,
		Pack: false,
	}
	_, err := h.ServeDNS(context.TODO(), &mwtest.ResponseWriter{}, r)
	if err != nil {
		t.Fatal(err)
		return
	}
}

type writer struct {
	queue []*tap.Message
}

func (w *writer) Write(b []byte) (int, error) {
	m := tap.Dnstap{}
	if err := proto.Unmarshal(b, &m); err != nil {
		return 0, err
	}
	if len(w.queue) == 0 {
		return 0, errors.New("message not expected")
	}
	if !test.MsgEqual(w.queue[0], m.Message) {
		return 0, fmt.Errorf("want: %v, have: %v", w.queue[0], m.Message)
	}
	w.queue = w.queue[1:]
	return len(b), nil
}

func TestDnstap(t *testing.T) {
	r := mwtest.Case{Qname: "example.org", Qtype: dns.TypeA}.Msg()
	resp := mwtest.Case{
		Qname: "example.org.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			mwtest.A("example.org. 3600	IN	A 10.0.0.1"),
		},
	}.Msg()
	tapr := msg.ToClientQuery(test.TestingData())
	tapresp := msg.ToClientResponse(test.TestingData())
	testCase(t, tapr, tapresp, r, resp)
}
