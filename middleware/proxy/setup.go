package proxy

import (
	"log"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/dnstap"

	"github.com/mholt/caddy"
)

func init() {
	caddy.RegisterPlugin("proxy", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	t := dnsserver.GetMiddleware(c, "trace")
	P := &Proxy{Trace: t}

	upstreams, err := NewStaticUpstreamsTap(&c.Dispenser, P)
	if err != nil {
		return middleware.Error("proxy", err)
	}

	if h := dnsserver.GetMiddleware(c, "dnstap"); h != nil {
		if d, ok := h.(dnstap.Dnstap); ok {
			P.Dnstap = d
		} else {
			// should it be fatal instead?
			log.Printf("[WARNING] Wrong type for dnstap middleware reference: %s", h)
		}
	}

	dnsserver.GetConfig(c).AddMiddleware(func(next middleware.Handler) middleware.Handler {
		P.Next = next
		P.Upstreams = &upstreams
		return P
	})

	c.OnStartup(OnStartupMetrics)

	for _, u := range upstreams {
		c.OnStartup(func() error {
			return u.Exchanger().OnStartup(P)
		})
		c.OnShutdown(func() error {
			return u.Exchanger().OnShutdown(P)
		})
		// Register shutdown handlers.
		c.OnShutdown(u.Stop)
	}

	return nil
}
