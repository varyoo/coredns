package dnstap

import (
	"github.com/coredns/coredns/middleware/dnstap/msg"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type ResponseWriter struct {
	dns.ResponseWriter
	log chan<- []byte
}

func (w *ResponseWriter) WriteMsg(resp *dns.Msg) error {
	state := request.Request{
		Req: resp,
		W:   w,
	}
	m, err := msg.NewClientResponse(&state, true)
	if err != nil {
		return errors.Wrap(err, "client response")
	}
	data, err := msg.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	w.log <- data
	return w.ResponseWriter.WriteMsg(resp)
}
