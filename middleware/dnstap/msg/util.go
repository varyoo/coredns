package msg

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
)

func parsePort(raw string) (uint32, error) {
	i, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.Wrap(err, "can't convert to int")
	}
	if i < 0 {
		return 0, errors.New("can't be < 0")
	}
	return uint32(i), nil
}

func ipPort(a net.Addr) (ip []byte, port uint32, err error) {
	rawip, rawport, err := net.SplitHostPort(a.String())
	if err != nil {
		err = errors.Wrap(err, "split ip:port")
		return
	}
	port, err = parsePort(rawport)
	if err != nil {
		err = errors.Wrap(err, "port")
		return
	}
	netip := net.ParseIP(rawip)
	if netip == nil {
		err = errors.Wrap(err, "ip")
		return
	}
	ip = []byte(netip)
	return
}
