package dnstap

import (
	"strconv"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap/out"

	"github.com/mholt/caddy"
	"github.com/pkg/errors"
)

func init() {
	caddy.RegisterPlugin("dnstap", caddy.Plugin{
		ServerType: "dns",
		Action:     wrapSetup,
	})
}

func wrapSetup(c *caddy.Controller) error {
	if err := setup(c); err != nil {
		return middleware.Error("dnstap", err)
	}
	return nil
}

func setup(c *caddy.Controller) error {
	c.Next() // 'dnstap'
	if !c.NextArg() {
		return c.ArgErr()
	}
	path := c.Val()
	if !c.NextArg() {
		return c.ArgErr()
	}
	pack, _ := strconv.ParseBool(c.Val())
	if c.NextArg() {
		return c.ArgErr()
	}

	dnstap := Dnstap{Pack: pack}

	o, err := out.NewSocket(path)
	if err != nil {
		return errors.Wrap(err, "output")
	}
	dnstap.Out = o

	c.OnShutdown(func() error {
		return errors.Wrap(o.Close(), "output")
	})

	dnsserver.GetConfig(c).AddMiddleware(
		func(next middleware.Handler) middleware.Handler {
			dnstap.Next = next
			return dnstap
		})

	return nil
}
