package dnstap

import (
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/dnstap/taprw"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// Dnstap is the dnstap handler.
type Dnstap struct {
	Next           plugin.Handler
	IO             IORoutine
	JoinRawMessage bool

	err error
}

type (
	// IORoutine is the dnstap I/O thread as defined by: <http://dnstap.info/Architecture>.
	IORoutine interface {
		Dnstap(tap.Dnstap)
	}
	// Tapper is implemented by the Context passed by the dnstap handler.
	Tapper interface {
		TapMessage(message *tap.Message, errorWhileBuildingMessage error)
		Pack() bool
	}
	tapContext struct {
		context.Context
		Dnstap
	}
)

// ContextKey defines the type of key that is used to save data into the context
type ContextKey string

const (
	// DnstapSendOption specifies the Dnstap message to be send.  Default is sent all.
	DnstapSendOption ContextKey = "dnstap-send-option"
)

// TapperFromContext will return a Tapper if the dnstap plugin is enabled.
func TapperFromContext(ctx context.Context) (t Tapper) {
	t, _ = ctx.(Tapper)
	return
}

// TapMessage implements Tapper.
func (h Dnstap) TapMessage(m *tap.Message, err error) {
	if err != nil {
		h.err = err
		return
	}

	t := tap.Dnstap_MESSAGE
	h.IO.Dnstap(tap.Dnstap{
		Type:    &t,
		Message: m,
	})
}

func (h Dnstap) Pack() bool {
	return h.JoinRawMessage
}

// ServeDNS logs the client query and response to dnstap and passes the dnstap Context.
func (h Dnstap) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	// Add send option into context so other plugin can decide on which DNSTap
	// message to be sent out
	sendOption := taprw.SendOption{Cq: true, Cr: true}
	newCtx := context.WithValue(ctx, DnstapSendOption, &sendOption)

	rw := &taprw.ResponseWriter{ResponseWriter: w, Tapper: &h, Query: r, Send: &sendOption}
	rw.SetQueryEpoch()

	code, err := plugin.NextOrFailure(h.Name(), h.Next, tapContext{newCtx, h}, rw, r)
	if err != nil {
		// ignore dnstap errors
		return code, err
	}

	if h.err != nil {
		return code, plugin.Error("dnstap", err)
	}

	return code, nil
}

// Name returns dnstap.
func (h Dnstap) Name() string { return "dnstap" }
