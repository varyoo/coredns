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

type Dnstap struct {
	Next middleware.Handler
	Out  io.Writer
	Pack bool
}

func (h Dnstap) TapMessage(m *tap.Message) error {
	frame, err := msg.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	_, err = h.Out.Write(frame)
	return err
}

func (h Dnstap) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	taprw := ResponseWriter{ResponseWriter: w, Taper: &h, Query: r, Pack: h.Pack}
	taprw.QueryEpoch()

	code, err := middleware.NextOrFailure(h.Name(), h.Next, ctx, taprw, r)
	if err != nil {
		return code, err
	}

	if err := DnstapError(taprw); err != nil {
		return code, err
	}

	return code, nil
}
func (h Dnstap) Name() string { return "dnstap" }
