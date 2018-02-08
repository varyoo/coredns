// Package taprw takes a query and intercepts the response.
// It will log both after the response is written.
package taprw

import (
	"time"

	"github.com/coredns/coredns/plugin/dnstap/msg"

	tap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

// SendOption stores the flag to indicate whether a certain DNSTap message to
// be sent out or not.
type SendOption struct {
	Cq bool
	Cr bool
}

// Tapper is what ResponseWriter needs to log to dnstap.
type Tapper interface {
	TapMessage(*tap.Message, error)
	Pack() bool
}

// ResponseWriter captures the client response and logs the query to dnstap.
// Single request use.
// SendOption configures Dnstap to selectively send Dnstap messages. Default is send all.
type ResponseWriter struct {
	QueryEpoch time.Time
	Query      *dns.Msg
	dns.ResponseWriter
	Tapper
	Send *SendOption
}

// WriteMsg writes back the response to the client and THEN works on logging the request
// and response to dnstap.
func (w *ResponseWriter) WriteMsg(resp *dns.Msg) (writeErr error) {
	writeErr = w.ResponseWriter.WriteMsg(resp)
	writeEpoch := time.Now()

	b := msg.New().Time(w.QueryEpoch)

	if w.Send == nil || w.Send.Cq {
		if w.Pack() {
			b.Msg(w.Query)
		}
		w.TapMessage(b.Addr(w.ResponseWriter.RemoteAddr()).ToClientQuery())
	}

	if w.Send == nil || w.Send.Cr {
		if writeErr == nil {
			if w.Pack() {
				b.Msg(resp)
			}
			w.TapMessage(b.Time(writeEpoch).ToClientResponse())
		}
	}

	return writeErr
}
