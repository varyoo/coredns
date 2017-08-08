package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/request"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

type dnsEx struct {
	Timeout time.Duration
	Options
}

// Options define the options understood by dns.Exchange.
type Options struct {
	ForceTCP bool // If true use TCP for upstream no matter what
	Dnstap
}

func newDNSEx() *dnsEx {
	return newDNSExWithOption(Options{})
}

func newDNSExWithOption(opt Options) *dnsEx {
	return &dnsEx{Timeout: defaultTimeout * time.Second, Options: opt}
}

func (d *dnsEx) Protocol() string          { return "dns" }
func (d *dnsEx) OnShutdown(p *Proxy) error { return nil }
func (d *dnsEx) OnStartup(p *Proxy) error  { return nil }

// Exchange implements the Exchanger interface.
// When *dns.Msg is not nil it is valid.
// When both *dns.Msg and error are not nil, the error should be reported.
func (d *dnsEx) Exchange(ctx context.Context, addr string, state request.Request) (*dns.Msg, error) {
	proto := state.Proto()
	if d.Options.ForceTCP {
		proto = "tcp"
	}
	co, err := net.DialTimeout(proto, addr, d.Timeout)
	if err != nil {
		return nil, err
	}

	// log the forwarded query to dnstap
	var taperr error
	b := msg.Builder{}
	if d.Options.Dnstap != nil {
		b.FromConn(co)
		b.Type = msg.OutsideQuery(tap.Message_FORWARDER_QUERY)
		b.Pack = state.Req.Pack
		taperr = d.Options.Tap(&b)
		if taperr != nil {
			taperr = fmt.Errorf("dnstap: %s", err)
		}
	}

	reply, _, err := d.ExchangeConn(state.Req, co)

	co.Close()

	if reply != nil && reply.Truncated {
		// Suppress proxy error for truncated responses
		err = nil
	}

	if err != nil {
		return nil, err
	}
	// Make sure it fits in the DNS response.
	reply, _ = state.Scrub(reply)
	reply.Compress = true
	reply.Id = state.Req.Id

	// log response to dnstap
	if d.Options.Dnstap != nil {
		b.Type = msg.OutsideResponse(tap.Message_FORWARDER_RESPONSE)
		b.Pack = reply.Pack
		if err := d.Options.Tap(&b); err != nil {
			return reply, fmt.Errorf("dnstap: %s", err)
		}
	}

	return reply, taperr
}

func (d *dnsEx) ExchangeConn(m *dns.Msg, co net.Conn) (*dns.Msg, time.Duration, error) {
	start := time.Now()
	r, err := exchange(m, co)
	rtt := time.Since(start)

	return r, rtt, err
}

func exchange(m *dns.Msg, co net.Conn) (*dns.Msg, error) {
	opt := m.IsEdns0()

	udpsize := uint16(dns.MinMsgSize)
	// If EDNS0 is used use that for size.
	if opt != nil && opt.UDPSize() >= dns.MinMsgSize {
		udpsize = opt.UDPSize()
	}

	dnsco := &dns.Conn{Conn: co, UDPSize: udpsize}

	writeDeadline := time.Now().Add(defaultTimeout)
	dnsco.SetWriteDeadline(writeDeadline)
	dnsco.WriteMsg(m)

	readDeadline := time.Now().Add(defaultTimeout)
	co.SetReadDeadline(readDeadline)
	r, err := dnsco.ReadMsg()

	dnsco.Close()
	if r == nil {
		return nil, err
	}
	return r, err
}
