package dnstap

import (
	"github.com/coredns/coredns/middleware"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

type Dnstap struct {
	Next middleware.Handler
}

func (h Dnstap) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return h.Next.ServeDNS(ctx, w, r)
}
func (h Dnstap) Name() string { return "dnstap" }
