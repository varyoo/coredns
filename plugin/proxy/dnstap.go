package proxy

import (
	"time"

	"github.com/coredns/coredns/plugin/dnstap"
	"github.com/coredns/coredns/plugin/dnstap/msg"
	"github.com/coredns/coredns/request"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func toDnstap(ctx context.Context, host string, ex Exchanger, state request.Request, reply *dns.Msg, start time.Time) {
	tapper := dnstap.TapperFromContext(ctx)
	if tapper == nil {
		return
	}

	// Query
	b := msg.New().Time(start).HostPort(host)

	t := ex.Transport()
	if t == "" {
		t = state.Proto()
	}
	if t == "tcp" {
		b.SocketProto = tap.SocketProtocol_TCP
	} else {
		b.SocketProto = tap.SocketProtocol_UDP
	}

	if tapper.Pack() {
		b.Msg(state.Req)
	}
	tapper.TapMessage(b.ToOutsideQuery(tap.Message_FORWARDER_QUERY))

	// Response
	if reply != nil {
		if tapper.Pack() {
			b.Msg(reply)
		}
		tapper.TapMessage(b.Time(time.Now()).
			ToOutsideResponse(tap.Message_FORWARDER_RESPONSE))
	}
}
