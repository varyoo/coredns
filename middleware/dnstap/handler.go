package dnstap

import (
	"golang.org/x/net/context"
	"io"

	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap/msg"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type ClientTap struct {
	Next middleware.Handler
	Out  io.Writer
	Pack bool
}

func (h ClientTap) TapMessage(m *tap.Message) error {
	frame, err := msg.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	_, err = h.Out.Write(frame)
	return err
}

func (h ClientTap) ServeDNS(ctx context.Context, w dns.ResponseWriter,
	r *dns.Msg) (int, error) {

	taprw := ResponseWriter{ResponseWriter: w, Tap: &h, Query: r, Pack: h.Pack}
	taprw.QueryEpoch()

	_, err := middleware.NextOrFailure(h.Name(), h.Next, ctx, taprw, r)
	return 0, err
}
func (h ClientTap) Name() string { return "dnstap" }
