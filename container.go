package main

import (
	"errors"
	"fmt"
	"strings"
)

var (
	PortNotMapped error = errors.New("port not mapped")
	EnvNotSet     error = errors.New("environment variable not set")
)

// Container describes a configured container.
type Container struct {
	// A cluster-unique ID.
	ID string
	// Environment variables.
	Environ []string
	// Ths container's addressable hostname.
	Hostname string
	// Mapped ports.
	Mappings []*Mapping
}

// Env retrieves the named environment variable. If not set an empty string is
// returned.
func (cntr *Container) Env(name string) string {
	for _, envVar := range cntr.Environ {
		parts := strings.SplitN(envVar, "=", 2)
		if parts[0] == name {
			if len(parts) == 1 {
				return ""
			} else {
				return parts[1]
			}
		}
	}
	return ""
}

// Mapping retrieves a port mapping from the container. If the port is not
// mapped the error `PortNotMapped` is returned.
func (cntr *Container) Mapping(port *Port) (*Address, error) {
	for _, m := range cntr.Mappings {
		if port.Equal(m.ContainerPort) {
			return m.HostAddress, nil
		}
	}
	return nil, PortNotMapped
}

// ContainerService describes how a container publishes access to a particular
// service.
type ContainerService struct {
	Name string
	Port *Port
}

// ParseContainerService converts a string of the form svc1:port/protocol and
// returns a ContainerService struct.
func ParseContainerService(service string) (*ContainerService, error) {
	parts := strings.SplitN(service, ":", 2)
	if len(parts) == 1 {
		return nil, fmt.Errorf("service must have a port %s", service)
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return nil, fmt.Errorf("service %s has invalid name %s", service, name)
	}
	portStr := strings.TrimSpace(parts[1])
	port, err := ParsePort(portStr)
	if err != nil {
		return nil, fmt.Errorf("service %s has invalid port %s", service, portStr)
	}
	return &ContainerService{name, port}, nil
}
