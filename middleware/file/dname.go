package file

import (
	"strings"

	"github.com/miekg/dns"
)

// substituteDNAME performs the DNAME substitution defined by RFC 6672,
// assuming the QTYPE of the query is not DNAME. It returns an empty
// string if there is no match.
func substituteDNAME(qname, owner, target string) string {
	if dns.IsSubDomain(owner, qname) && qname != owner {
		labels := dns.SplitDomainName(qname)
		labels = append(labels[0:len(labels)-dns.CountLabel(owner)], dns.SplitDomainName(target)...)

		return strings.Join(labels, ".") + "."
	}

	return ""
}
