package dnstap

import (
	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/request"
	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"log"
)

type Tap interface {
	TapMessage(m *tap.Message) error
}

type ResponseWriter struct {
	queryData msg.Data
	Query     *dns.Msg
	dns.ResponseWriter
	Tap
	Pack bool
}

func (w ResponseWriter) QueryEpoch() {
	msg.Epoch(&w.queryData)
}

// Write back the response to the client and THEN work on logging the request
// and response to dnstap.
// Not sure on how to best report errors with this approch.
func (w ResponseWriter) WriteMsg(resp *dns.Msg) error {
	writeErr := w.ResponseWriter.WriteMsg(resp)

	if err := w.tapQuery(); err != nil {
		log.Printf("[ERROR] can't log client query: %s", err)
	}

	if err := w.tapResponse(resp); err != nil {
		log.Printf("[ERROR] can't log client response: %s", err)
	}

	return writeErr
}
func (w ResponseWriter) tapQuery() error {
	w.queryData.Type = tap.Message_CLIENT_QUERY
	req := request.Request{W: w, Req: w.Query}
	if err := msg.FromRequest(&w.queryData, req); err != nil {
		return err
	}
	if w.Pack {
		if err := msg.Pack(&w.queryData, w.Query); err != nil {
			return errors.Wrap(err, "pack")
		}
	}
	return errors.Wrap(
		w.Tap.TapMessage(msg.ToMsg(&w.queryData)),
		"tap",
	)
}
func (w ResponseWriter) tapResponse(resp *dns.Msg) error {
	d := &msg.Data{}
	msg.Epoch(d)
	d.Type = tap.Message_CLIENT_RESPONSE
	req := request.Request{W: w, Req: resp}
	if err := msg.FromRequest(d, req); err != nil {
		return err
	}
	if w.Pack {
		if err := msg.Pack(d, resp); err != nil {
			return errors.Wrap(err, "pack")
		}
	}
	return errors.Wrap(
		w.Tap.TapMessage(msg.ToMsg(d)),
		"tap",
	)
}
