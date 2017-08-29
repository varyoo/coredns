package dnstap

import (
	"fmt"
	"io"
	"log"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap/out"
	"github.com/coredns/coredns/middleware/pkg/dnsutil"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyfile"
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

func parseConfig(c *caddyfile.Dispenser) (path string, socket, full bool, err error) {
	c.Next() // directive name

	if !c.Args(&path) {
		err = c.ArgErr()
		return
	}

	servers, err := dnsutil.ParseHostPortOrFile(path)
	if err != nil {
		socket = true
		err = nil
	} else {
		path = servers[0]
	}

	full = c.NextArg() && c.Val() == "full"

	return
}

func setup(c *caddy.Controller) error {
	path, socket, full, err := parseConfig(&c.Dispenser)
	if err != nil {
		return err
	}

	dnstap := Dnstap{Pack: full}

	var o io.WriteCloser
	if socket {
		o, err = out.NewSocket(path)
		if err != nil {
			log.Printf("[WARN] Can't connect to %s at the moment", path)
		}
	} else {
		o = out.NewTCP(path)
	}
	dnstap.Out = o

	c.OnShutdown(func() error {
		if err := o.Close(); err != nil {
			return fmt.Errorf("output: %s", err)
		}
		return nil
	})

	dnsserver.GetConfig(c).AddMiddleware(
		func(next middleware.Handler) middleware.Handler {
			dnstap.Next = next
			return dnstap
		})

	return nil
}
