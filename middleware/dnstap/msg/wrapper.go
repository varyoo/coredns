package msg

import (
	lib "github.com/dnstap/golang-dnstap"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func wrap(m *lib.Message) lib.Dnstap {
	t := lib.Dnstap_MESSAGE
	w := lib.Dnstap{
		Type:    &t,
		Message: m,
	}
	return w
}

func Marshal(m *Msg) (data []byte, err error) {
	event := wrap(&m.Message)
	data, err = proto.Marshal(&event)
	if err != nil {
		err = errors.Wrap(err, "proto")
		return
	}
	return
}
