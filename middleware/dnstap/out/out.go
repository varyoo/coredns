package out

import (
	fs "github.com/farsightsec/golang-framestream"
	"github.com/pkg/errors"
	"io"
)

var FSContentType = []byte("protobuf:dnstap.Dnstap")

type Output struct {
	enc *fs.Encoder
}

func NewOutput(w io.Writer) (*Output, error) {
	enc, err := fs.NewEncoder(w, &fs.EncoderOptions{ContentType: FSContentType})
	if err != nil {
		return nil, errors.Wrap(err, "framestream")
	}
	return &Output{enc}, nil
}

func (o *Output) Write(frame []byte) (int, error) {
	return o.enc.Write(frame)
}
func (o *Output) Close() {
	o.enc.Flush()
	o.enc.Close()
}
