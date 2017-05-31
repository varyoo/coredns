package dnstap

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	"github.com/mholt/caddy"
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

	dnsserver.GetConfig(c).AddMiddleware(func(next middleware.Handler) middleware.Handler {
		return Dnstap{Next: next}
	})

	return nil
}
