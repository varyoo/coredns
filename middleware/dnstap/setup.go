package dnstap

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap/out"
	"github.com/mholt/caddy"
	"github.com/pkg/errors"
	"os"
	"strconv"
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

	clientTap := ClientTap{Pack: pack}

	w, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "create db")
	}
	o, err := out.NewOutput(w)
	if err != nil {
		return errors.Wrap(err, "output")
	}
	clientTap.Out = o

	c.OnShutdown(func() error {
		o.Close()
		return nil
	})

	dnsserver.GetConfig(c).AddMiddleware(
		func(next middleware.Handler) middleware.Handler {
			clientTap.Next = next
			return clientTap
		})

	return nil
}
