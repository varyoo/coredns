package dnstap

import (
	"github.com/coredns/coredns/middleware"
	lib "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

type Dnstap struct {
	out  *lib.FrameStreamOutput
	Next middleware.Handler
}

func (h Dnstap) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	catch := ResponseWriter{w, h.out.GetOutputChannel()}
	_, err := middleware.NextOrFailure(h.Name(), h.Next, ctx, &catch, r)
	return 0, err
}
func (h Dnstap) Name() string { return "dnstap" }
