package forward

import (
	"context"
	"net"
	"time"

	"github.com/coredns/coredns/plugin/dnstap"
	"github.com/coredns/coredns/plugin/dnstap/msg"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

func logToDnstap(ctx context.Context, addr net.Addr, req, ret *dns.Msg, reqTime time.Time) error {
	t := dnstap.TapperFromContext(ctx)
	if t == nil {
		return nil
	}

	b := msg.New().Time(reqTime).Addr(addr)

	if t.Pack() {
		b.Msg(req)
	}

	m, err := b.ToOutsideQuery(tap.Message_FORWARDER_QUERY)
	if err != nil {
		return err
	}

	// log the query
	t.TapMessage(m)

	if ret != nil {
		if t.Pack() {
			b.Msg(ret)
		}

		m, err := b.Time(time.Now()).ToOutsideResponse(tap.Message_FORWARDER_RESPONSE)
		if err != nil {
			return err
		}

		// log the optional response
		t.TapMessage(m)
	}

	return nil
}
