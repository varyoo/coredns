package file

import (
	"sort"
	"strings"
	"testing"

	"github.com/coredns/coredns/middleware/pkg/dnsrecorder"
	"github.com/coredns/coredns/middleware/test"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

var dnameTestCases = []test.Case{
	{
		Qname: "dname.miek.nl.", Qtype: dns.TypeDNAME,
		Answer: []dns.RR{
			test.DNAME("dname.miek.nl.	1800	IN	DNAME	test.miek.nl."),
		},
		Ns: miekAuth,
	},
}

func TestLookupDNAME(t *testing.T) {
	zone, err := Parse(strings.NewReader(dbMiekNLDNAME), testzone, "stdin")
	if err != nil {
		t.Fatalf("Expect no error when reading zone, got %q", err)
	}

	fm := File{Next: test.ErrorHandler(), Zones: Zones{Z: map[string]*Zone{testzone: zone}, Names: []string{testzone}}}
	ctx := context.TODO()

	for _, tc := range dnameTestCases {
		m := tc.Msg()

		rec := dnsrecorder.New(&test.ResponseWriter{})
		_, err := fm.ServeDNS(ctx, rec, m)
		if err != nil {
			t.Errorf("Expected no error, got %v\n", err)
			return
		}

		resp := rec.Msg
		sort.Sort(test.RRSet(resp.Answer))
		sort.Sort(test.RRSet(resp.Ns))
		sort.Sort(test.RRSet(resp.Extra))

		if !test.Header(t, tc, resp) {
			t.Logf("%v\n", resp)
			continue
		}
		if !test.Section(t, tc, test.Answer, resp.Answer) {
			t.Logf("%v\n", resp)
		}
		if !test.Section(t, tc, test.Ns, resp.Ns) {
			t.Logf("%v\n", resp)
		}
		if !test.Section(t, tc, test.Extra, resp.Extra) {
			t.Logf("%v\n", resp)
		}
	}
}

const dbMiekNLDNAME = `
$TTL    30M
$ORIGIN miek.nl.
@       IN      SOA     linode.atoom.net. miek.miek.nl. (
                             1282630057 ; Serial
                             4H         ; Refresh
                             1H         ; Retry
                             7D         ; Expire
                             4H )       ; Negative Cache TTL
                IN      NS      linode.atoom.net.
                IN      NS      ns-ext.nlnetlabs.nl.
                IN      NS      omval.tednet.nl.
                IN      NS      ext.ns.whyscream.net.

test            IN      MX      1  aspmx.l.google.com.
                IN      MX      5  alt1.aspmx.l.google.com.
                IN      MX      5  alt2.aspmx.l.google.com.
                IN      MX      10 aspmx2.googlemail.com.
                IN      MX      10 aspmx3.googlemail.com.
a.test          IN      A       139.162.196.78
                IN      AAAA    2a01:7e00::f03c:91ff:fef1:6735
www.test        IN      CNAME   a.test

dname           IN      DNAME   test`
