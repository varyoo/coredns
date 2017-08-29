package dnstap

import (
	"github.com/mholt/caddy"
	"testing"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		file   string
		path   string
		full   bool
		socket bool
		fail   bool
	}{
		{"dnstap dnstap.sock full", "dnstap.sock", true, true, false},
		{"dnstap dnstap.sock", "dnstap.sock", false, true, false},
		{"dnstap 127.0.0.1:6000", "127.0.0.1:6000", false, false, false},
		{"dnstap", "fail", false, true, true},
	}
	for _, c := range tests {
		cad := caddy.NewTestController("dns", c.file)
		path, socket, full, err := parseConfig(&cad.Dispenser)
		if c.fail {
			if err == nil {
				t.Errorf("%s: %s", c.file, err)
			}
		} else if err != nil || path != c.path || full != c.full || socket != c.socket {
			t.Errorf("expected: %v\nhave: %s, %s, %b, %b\n", c, err, path, full, socket)
		}
	}
}
