package dnstap

import (
	"golang.org/x/net/context"
	"io"

	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/middleware/dnstap/taprw"

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
	rw := &taprw.ResponseWriter{ResponseWriter: w, Taper: &h, Query: r, Pack: h.Pack}
	rw.QueryEpoch()

	code, err := middleware.NextOrFailure(h.Name(), h.Next, ctx, rw, r)
	if err != nil {
		// ignore dnstap errors
		return code, err
	}

	if err := taprw.DnstapError(rw); err != nil {
		return code, err
	}

	return code, nil
}
func (h Dnstap) Name() string { return "dnstap" }
