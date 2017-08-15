// Package taprw takes a query and intercepts the response.
// It will log both after the response is written.
package taprw

import (
	"fmt"

	"github.com/coredns/coredns/middleware/dnstap/msg"

	"github.com/miekg/dns"
)

type Taper interface {
	Tap(msg.Message) error
}

// Single request use.
type ResponseWriter struct {
	builder msg.Builder
	Query   *dns.Msg
	dns.ResponseWriter
	Taper
	Pack bool
	err  error
}

// Check if a dnstap error occurred.
// Set during ResponseWriter.Write.
func (w ResponseWriter) DnstapError() error {
	return w.err
}

// To be called as soon as possible.
func (w *ResponseWriter) QueryEpoch() {
	w.builder.Epoch()
}

// Write back the response to the client and THEN work on logging the request
// and response to dnstap.
// Dnstap errors to be checked by DnstapError.
func (w *ResponseWriter) WriteMsg(resp *dns.Msg) error {
	writeErr := w.ResponseWriter.WriteMsg(resp)
	writeSec := msg.Epoch()

	b := &w.builder
	b.Type = msg.ClientQuery
	b.RemoteAddr = w.ResponseWriter.RemoteAddr()
	b.Pack = w.Query.Pack
	if err := w.Taper.Tap(b); err != nil {
		w.err = fmt.Errorf("client query: %s", err)
		// don't forget to call DnstapError later
	}

	if writeErr == nil {
		b.Type = msg.ClientResponse
		b.Pack = resp.Pack
		b.TimeSec = writeSec
		if err := w.Taper.Tap(b); err != nil {
			w.err = fmt.Errorf("client response: %s", err)
		}
	}

	return writeErr
}
