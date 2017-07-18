package proxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coredns/coredns/middleware"
	"github.com/coredns/coredns/middleware/pkg/dnsutil"
	"github.com/coredns/coredns/middleware/pkg/tls"
	"github.com/mholt/caddy/caddyfile"
	"github.com/miekg/dns"
)

var (
	supportedPolicies = make(map[string]func() Policy)
)

type staticUpstream struct {
	from string
	stop chan struct{}  // Signals running goroutines to stop.
	wg   sync.WaitGroup // Used to wait for running goroutines to stop.

	Hosts  HostPool
	Policy Policy
	Spray  Policy

	FailTimeout time.Duration
	MaxFails    int32
	Future      time.Duration
	HealthCheck struct {
		Path     string
		Port     string
		Interval time.Duration
	}
	WithoutPathPrefix string
	IgnoredSubDomains []string
	ex                Exchanger
}

// NewStaticUpstreams parses the configuration input and sets up
// static upstreams for the proxy middleware.
func NewStaticUpstreams(c *caddyfile.Dispenser) ([]Upstream, error) {
	return NewStaticUpstreamsTap(c, nil)
}

func NewStaticUpstreamsTap(c *caddyfile.Dispenser, taper Taper) ([]Upstream, error) {
	var upstreams []Upstream
	for c.Next() {
		ex := newDNSEx()
		if taper != nil {
			ex.Taper = taper
		}
		upstream := &staticUpstream{
			from:        ".",
			stop:        make(chan struct{}),
			Hosts:       nil,
			Policy:      &Random{},
			Spray:       nil,
			FailTimeout: 10 * time.Second,
			MaxFails:    1,
			Future:      60 * time.Second,
			ex:          ex,
		}

		if !c.Args(&upstream.from) {
			return upstreams, c.ArgErr()
		}
		to := c.RemainingArgs()
		if len(to) == 0 {
			return upstreams, c.ArgErr()
		}

		// process the host list, substituting in any nameservers in files
		toHosts, err := dnsutil.ParseHostPortOrFile(to...)
		if err != nil {
			return upstreams, err
		}

		for c.NextBlock() {
			if err := parseBlock(c, upstream); err != nil {
				return upstreams, err
			}
		}

		upstream.Hosts = make([]*UpstreamHost, len(toHosts))
		for i, host := range toHosts {
			uh := &UpstreamHost{
				Name:        host,
				Conns:       0,
				Fails:       0,
				FailTimeout: upstream.FailTimeout,

				CheckDown: func(upstream *staticUpstream) UpstreamHostDownFunc {
					return func(uh *UpstreamHost) bool {

						down := false

						uh.checkMu.Lock()
						until := uh.OkUntil
						uh.checkMu.Unlock()

						if !until.IsZero() && time.Now().After(until) {
							down = true
						}

						fails := atomic.LoadInt32(&uh.Fails)
						if fails >= upstream.MaxFails && upstream.MaxFails != 0 {
							down = true
						}
						return down
					}
				}(upstream),
				WithoutPathPrefix: upstream.WithoutPathPrefix,
			}

			upstream.Hosts[i] = uh
		}

		if upstream.HealthCheck.Path != "" {
			upstream.wg.Add(1)
			go func() {
				defer upstream.wg.Done()
				upstream.HealthCheckWorker(upstream.stop)
			}()
		}
		upstreams = append(upstreams, upstream)
	}
	return upstreams, nil
}

// Stop sends a signal to all goroutines started by this staticUpstream to exit
// and waits for them to finish before returning.
func (u *staticUpstream) Stop() error {
	close(u.stop)
	u.wg.Wait()
	return nil
}

// RegisterPolicy adds a custom policy to the proxy.
func RegisterPolicy(name string, policy func() Policy) {
	supportedPolicies[name] = policy
}

func (u *staticUpstream) From() string {
	return u.from
}

func parseBlock(c *caddyfile.Dispenser, u *staticUpstream) error {
	switch c.Val() {
	case "policy":
		if !c.NextArg() {
			return c.ArgErr()
		}
		policyCreateFunc, ok := supportedPolicies[c.Val()]
		if !ok {
			return c.ArgErr()
		}
		u.Policy = policyCreateFunc()
	case "fail_timeout":
		if !c.NextArg() {
			return c.ArgErr()
		}
		dur, err := time.ParseDuration(c.Val())
		if err != nil {
			return err
		}
		u.FailTimeout = dur
	case "max_fails":
		if !c.NextArg() {
			return c.ArgErr()
		}
		n, err := strconv.Atoi(c.Val())
		if err != nil {
			return err
		}
		u.MaxFails = int32(n)
	case "health_check":
		if !c.NextArg() {
			return c.ArgErr()
		}
		var err error
		u.HealthCheck.Path, u.HealthCheck.Port, err = net.SplitHostPort(c.Val())
		if err != nil {
			return err
		}
		u.HealthCheck.Interval = 30 * time.Second
		if c.NextArg() {
			dur, err := time.ParseDuration(c.Val())
			if err != nil {
				return err
			}
			u.HealthCheck.Interval = dur
			u.Future = 2 * dur

			// set a minimum of 3 seconds
			if u.Future < (3 * time.Second) {
				u.Future = 3 * time.Second
			}
		}
	case "without":
		if !c.NextArg() {
			return c.ArgErr()
		}
		u.WithoutPathPrefix = c.Val()
	case "except":
		ignoredDomains := c.RemainingArgs()
		if len(ignoredDomains) == 0 {
			return c.ArgErr()
		}
		for i := 0; i < len(ignoredDomains); i++ {
			ignoredDomains[i] = strings.ToLower(dns.Fqdn(ignoredDomains[i]))
		}
		u.IgnoredSubDomains = ignoredDomains
	case "spray":
		u.Spray = &Spray{}
	case "protocol":
		encArgs := c.RemainingArgs()
		if len(encArgs) == 0 {
			return c.ArgErr()
		}
		switch encArgs[0] {
		case "dns":
			if len(encArgs) > 1 {
				if encArgs[1] == "force_tcp" {
					opts := Options{ForceTCP: true}
					u.ex = newDNSExWithOption(opts)
				} else {
					return fmt.Errorf("only force_tcp allowed as parameter to dns")
				}
			} else {
				u.ex = newDNSEx()
			}
		case "https_google":
			boot := []string{"8.8.8.8:53", "8.8.4.4:53"}
			if len(encArgs) > 2 && encArgs[1] == "bootstrap" {
				boot = encArgs[2:]
			}

			u.ex = newGoogle("", boot) // "" for default in google.go
		case "grpc":
			if len(encArgs) == 2 && encArgs[1] == "insecure" {
				u.ex = newGrpcClient(nil, u)
				return nil
			}
			tls, err := tls.NewTLSConfigFromArgs(encArgs[1:]...)
			if err != nil {
				return err
			}
			u.ex = newGrpcClient(tls, u)
		default:
			return fmt.Errorf("%s: %s", errInvalidProtocol, encArgs[0])
		}

	default:
		return c.Errf("unknown property '%s'", c.Val())
	}
	return nil
}

// This was moved into a thread so that each host could throw a health
// check at the same time.  The reason for this is that if we are checking
// 3 hosts, and the first one is gone, and we spend minutes timing out to
// fail it, we would not have been doing any other health checks in that
// time.  So we now have a per-host lock and a threaded health check.
//
// We use the Checking bool to avoid concurrent checks against the same
// host; if one is taking a long time, the next one will find a check in
// progress and simply return before trying.
//
// We are carefully avoiding having the mutex locked while we check,
// otherwise checks will back up, potentially a lot of them if a host is
// absent for a long time.  This arrangement makes checks quickly see if
// they are the only one running and abort otherwise.
func healthCheckUrl(nextTs time.Time, host *UpstreamHost) {

	// lock for our bool check.  We don't just defer the unlock because
	// we don't want the lock held while http.Get runs
	host.checkMu.Lock()

	// are we mid check?  Don't run another one
	if host.Checking {
		host.checkMu.Unlock()
		return
	}

	host.Checking = true
	host.checkMu.Unlock()

	//log.Printf("[DEBUG] Healthchecking %s, nextTs is %s\n", url, nextTs.Local())

	// fetch that url.  This has been moved into a go func because
	// when the remote host is not merely not serving, but actually
	// absent, then tcp syn timeouts can be very long, and so one
	// fetch could last several check intervals
	if r, err := http.Get(host.CheckUrl); err == nil {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()

		if r.StatusCode < 200 || r.StatusCode >= 400 {
			log.Printf("[WARNING] Host %s health check returned HTTP code %d\n",
				host.Name, r.StatusCode)
			nextTs = time.Unix(0, 0)
		}
	} else {
		log.Printf("[WARNING] Host %s health check probe failed: %v\n", host.Name, err)
		nextTs = time.Unix(0, 0)
	}

	host.checkMu.Lock()
	host.Checking = false
	host.OkUntil = nextTs
	host.checkMu.Unlock()
}

func (u *staticUpstream) healthCheck() {
	for _, host := range u.Hosts {

		if host.CheckUrl == "" {
			var hostName, checkPort string

			// The DNS server might be an HTTP server.  If so, extract its name.
			ret, err := url.Parse(host.Name)
			if err == nil && len(ret.Host) > 0 {
				hostName = ret.Host
			} else {
				hostName = host.Name
			}

			// Extract the port number from the parsed server name.
			checkHostName, checkPort, err := net.SplitHostPort(hostName)
			if err != nil {
				checkHostName = hostName
			}

			if u.HealthCheck.Port != "" {
				checkPort = u.HealthCheck.Port
			}

			host.CheckUrl = "http://" + net.JoinHostPort(checkHostName, checkPort) + u.HealthCheck.Path
		}

		// calculate this before the get
		nextTs := time.Now().Add(u.Future)

		// locks/bools should prevent requests backing up
		go healthCheckUrl(nextTs, host)
	}
}

func (u *staticUpstream) HealthCheckWorker(stop chan struct{}) {
	ticker := time.NewTicker(u.HealthCheck.Interval)
	u.healthCheck()
	for {
		select {
		case <-ticker.C:
			u.healthCheck()
		case <-stop:
			ticker.Stop()
			return
		}
	}
}

func (u *staticUpstream) Select() *UpstreamHost {
	pool := u.Hosts
	if len(pool) == 1 {
		if pool[0].Down() && u.Spray == nil {
			return nil
		}
		return pool[0]
	}
	allDown := true
	for _, host := range pool {
		if !host.Down() {
			allDown = false
			break
		}
	}
	if allDown {
		if u.Spray == nil {
			return nil
		}
		return u.Spray.Select(pool)
	}

	if u.Policy == nil {
		h := (&Random{}).Select(pool)
		if h != nil {
			return h
		}
		if h == nil && u.Spray == nil {
			return nil
		}
		return u.Spray.Select(pool)
	}

	h := u.Policy.Select(pool)
	if h != nil {
		return h
	}

	if u.Spray == nil {
		return nil
	}
	return u.Spray.Select(pool)
}

func (u *staticUpstream) IsAllowedDomain(name string) bool {
	if dns.Name(name) == dns.Name(u.From()) {
		return true
	}

	for _, ignoredSubDomain := range u.IgnoredSubDomains {
		if middleware.Name(ignoredSubDomain).Matches(name) {
			return false
		}
	}
	return true
}

func (u *staticUpstream) Exchanger() Exchanger { return u.ex }
