package docker

import (
	"github.com/BlueDragonX/beacon/beacon"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

// parsePort takes a port of the form "80/tcp" and returns the port number and protocol.
func parsePort(port string) (number int, protocol beacon.Protocol, err error) {
	parts := strings.SplitN(port, "/", 2)
	if len(parts) == 1 {
		protocol = beacon.TCP
	} else {
		switch parts[1] {
		case "tcp":
			protocol = beacon.TCP
		case "udp":
			protocol = beacon.UDP
		default:
			err = errors.Errorf("unsupported protocol %s", parts[1])
			return
		}
	}
	number, err = strconv.Atoi(parts[0])
	if err != nil {
		err = errors.Wrapf(err, "unable to parse port number %s", parts[0])
	}
	return
}
