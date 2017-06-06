package dnstap

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	lib "github.com/dnstap/golang-dnstap"
	"github.com/mholt/caddy"
	"github.com/pkg/errors"
)

func init() {
	caddy.RegisterPlugin("dnstap", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}
func setup(c *caddy.Controller) error {
	c.Next() // 'dnstap'
	if c.NextArg() {
		return middleware.Error("dnstap", c.ArgErr())
	}

	tap := Dnstap{}
	out, err := lib.NewFrameStreamOutputFromFilename("/tmp/db")
	if err != nil {
		return errors.Wrap(err, "output")
	}
	tap.out = out

	go out.RunOutputLoop()

	c.OnShutdown(func() error {
		out.Close()
		return nil
	})

	dnsserver.GetConfig(c).AddMiddleware(func(next middleware.Handler) middleware.Handler {
		tap.Next = next
		return tap
	})

	return nil
}
