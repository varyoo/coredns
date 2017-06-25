// Package taprw takes a query and intercepts the response.
// It will log both after the response is written.
package taprw

import (
	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/request"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type Taper interface {
	TapMessage(m *tap.Message) error
}

type ResponseWriter struct {
	queryData msg.Data
	Query     *dns.Msg
	dns.ResponseWriter
	Taper
	Pack bool
	err  error
}

// Check if a dnstap error occured.
// Set during ResponseWriter.Write.
func DnstapError(w *ResponseWriter) error {
	return w.err
}

// To be called as soon as possible.
func (w *ResponseWriter) QueryEpoch() {
	msg.Epoch(&w.queryData)
}

// Write back the response to the client and THEN work on logging the request
// and response to dnstap.
// Dnstap errors to be checked by DnstapError.
func (w *ResponseWriter) WriteMsg(resp *dns.Msg) error {
	writeErr := w.ResponseWriter.WriteMsg(resp)

	if err := tapQuery(w); err != nil {
		w.err = errors.Wrap(err, "can't log client query")
		// don't forget to call DnstapError later
	}

	if writeErr == nil {
		if err := tapResponse(w, resp); err != nil {
			w.err = errors.Wrap(err, "can't log client response")
		}
	}

	return writeErr
}
func tapQuery(w *ResponseWriter) error {
	req := request.Request{W: w.ResponseWriter, Req: w.Query}
	if err := msg.FromRequest(&w.queryData, req); err != nil {
		return err
	}
	if w.Pack {
		if err := msg.Pack(&w.queryData, w.Query); err != nil {
			return errors.Wrap(err, "pack")
		}
	}
	return w.Taper.TapMessage(msg.ToClientQuery(&w.queryData))
}
func tapResponse(w *ResponseWriter, resp *dns.Msg) error {
	d := &msg.Data{}
	msg.Epoch(d)
	req := request.Request{W: w, Req: resp}
	if err := msg.FromRequest(d, req); err != nil {
		return err
	}
	if w.Pack {
		if err := msg.Pack(d, resp); err != nil {
			return errors.Wrap(err, "pack")
		}
	}
	return w.Taper.TapMessage(msg.ToClientResponse(d))
}
