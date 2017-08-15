package dnstap

import (
	"fmt"
	"io"

	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/middleware/dnstap/taprw"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

type Dnstap struct {
	Next middleware.Handler
	Out  io.Writer
	Pack bool
}

type (
	// Taper is implemented by the Context passed by the dnstap handler.
	Taper interface {
		Tap(msg.Message) error
	}
	tapContext struct {
		context.Context
		Dnstap
	}
)

// TaperFromContext will return a Taper if the dnstap middleware is enabled.
func TaperFromContext(ctx context.Context) (t Taper) {
	t, _ = ctx.(Taper)
	return
}

func tapMessageTo(w io.Writer, m *tap.Message) error {
	frame, err := msg.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal: %s", err)
	}
	_, err = w.Write(frame)
	return err
}

func (h Dnstap) TapMessage(m *tap.Message) error {
	return tapMessageTo(h.Out, m)
}

func (h Dnstap) Tap(b msg.Message) error {
	m, err := b.Message(h.Pack)
	if err != nil {
		return err
	}
	return tapMessageTo(h.Out, m)
}

func (h Dnstap) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	rw := &taprw.ResponseWriter{ResponseWriter: w, Taper: &h, Query: r, Pack: h.Pack}
	rw.QueryEpoch()

	code, err := middleware.NextOrFailure(h.Name(), h.Next, tapContext{ctx, h}, rw, r)
	if err != nil {
		// ignore dnstap errors
		return code, err
	}

	if err := rw.DnstapError(); err != nil {
		return code, middleware.Error("dnstap", err)
	}

	return code, nil
}
func (h Dnstap) Name() string { return "dnstap" }
