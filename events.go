package main

import (
	"fmt"
)

const (
	// Event states.
	Add       = "add"
	Remove    = "remove"
	Update    = "update"
	Heartbeat = "heartbeat"
)

type ContainerEvent struct {
	State       string
	ContainerId string
}

type ServiceEvent struct {
	State   string
	Service *Service
}

func (e ServiceEvent) String() string {
	return fmt.Sprintf("%v %s", e.State, e.Service)
}
