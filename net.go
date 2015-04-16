package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Port describe a port number and protocol.
type Port struct {
	Number   int
	Protocol string
}

// ParsePort takes a string of the format 'number/protocol' and converts it to a Port.
func ParsePort(portStr string) (*Port, error) {
	var err error
	port := &Port{}
	parts := strings.SplitN(strings.TrimSpace(portStr), "/", 2)
	if port.Number, err = strconv.Atoi(parts[0]); err != nil {
		return nil, fmt.Errorf("port number is invalid %s", parts[0])
	}

	if len(parts) == 1 {
		port.Protocol = "tcp"
	} else {
		port.Protocol = strings.ToLower(parts[1])
		if port.Protocol != "tcp" && port.Protocol != "udp" {
			return nil, fmt.Errorf("port has unrecognized protocol %s", port.Protocol)
		}
	}
	return port, nil
}

// Equal checks for equality between two ports.
func (left *Port) Equal(right *Port) bool {
	return left.Number == right.Number && left.Protocol == right.Protocol
}

// String converts the port to its string representation.
func (p *Port) String() string {
	return fmt.Sprintf("%d/%s", p.Number, p.Protocol)
}

// Address is a hostname/port pairing.
type Address struct {
	Hostname string
	Port     *Port
}

// ParseAddress takes a string of the form host:port/protocol and converts it
// to an Address.
func ParseAddress(addr string) (*Address, error) {
	parts := strings.SplitN(addr, ":", 2)
	if len(parts) == 1 {
		return nil, fmt.Errorf("address must have a port %s", addr)
	}
	port, err := ParsePort(parts[1])
	if err != nil {
		return nil, err
	}
	return &Address{parts[0], port}, nil
}

// Equal returns true if the addresses are equal.
func (left *Address) Equal(right *Address) bool {
	return left.Hostname == right.Hostname && left.Port.Equal(right.Port)
}

// String converts the address to its string representation.
func (a *Address) String() string {
	return fmt.Sprintf("%s:%s", a.Hostname, a.Port)
}

// StringNoProtocol converts the address to its string representation without
// the protocol part.
func (a *Address) StringNoProtocol() string {
	return fmt.Sprintf("%s:%d", a.Hostname, a.Port.Number)
}

// Mapping describes a port mapped from the host to a container.
type Mapping struct {
	HostAddress   *Address
	ContainerPort *Port
}
