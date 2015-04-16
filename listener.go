package main

import (
	"io"
)

type ContainerAction int

const (
	// Container event actions.
	ContainerAdd ContainerAction = iota
	ContainerRemove
)

// ContainerEvent is used by the Listener to notify Beacon of
// container actions.
type ContainerEvent struct {
	Action    ContainerAction
	Container *Container
}

// Listener emits container events retrieved from a container runtime.
//
// Listen queues container events on the provided channel.
// Close stops the listener and cleans up.
type Listener interface {
	Listen(chan<- *ContainerEvent)
	io.Closer
}
