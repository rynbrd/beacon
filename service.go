package main

import (
	"errors"
	"fmt"
	"github.com/samalba/dockerclient"
	"strings"
)

// Describes a service mapping from container to host.
type Service struct {
	Name          string
	ContainerId   string
	ContainerName string
	ContainerPort int
	HostName      string
	HostPort      int
	Protocol      string
}

// Parse a config value into service name and port/protocol.
func (svc *Service) loadConfig(config string) (err error) {
	parts := strings.SplitN(strings.TrimSpace(config), ":", 2)
	if len(parts) != 2 {
		err = errors.New(fmt.Sprintf("invalid service config: %s", config))
		return
	}
	svc.Name = strings.TrimSpace(parts[0])
	svc.ContainerPort, svc.Protocol, err = parsePort(parts[1])
	return
}

// Load the container details and port bindings.
func (svc *Service) loadInfo(containerInfo *dockerclient.ContainerInfo, defaultHostname string) error {
	key := fmt.Sprintf("%v/%v", svc.ContainerPort, svc.Protocol)
	bindings, ok := containerInfo.HostConfig.PortBindings[key]
	if !ok || len(bindings) == 0 {
		return errors.New(fmt.Sprintf("service not exposed: %v", key))
	}
	binding := bindings[0]
	hostPort, _, err := parsePort(binding.HostPort)
	if err != nil {
		return err
	}
	svc.ContainerId = containerInfo.Id
	svc.ContainerName = containerInfo.Config.Hostname
	svc.HostName = binding.HostIp
	svc.HostPort = hostPort
	if svc.HostName == "0.0.0.0" && defaultHostname != "" {
		svc.HostName = defaultHostname
	}
	return nil
}

// Return a string hash to uniquely identify the service.
func (svc *Service) Hash() string {
	return fmt.Sprintf("%v|%v", svc.Name, svc.ContainerId)
}

// Return a string representation of the service.
func (svc *Service) String() string {
	return fmt.Sprintf("%v ( %v:%v -> %v:%v/%v )", svc.Name, svc.HostName,
		svc.HostPort, svc.ContainerName, svc.ContainerPort, svc.Protocol)
}
