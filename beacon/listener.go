package beacon

import (
	"github.com/BlueDragonX/beacon/container"
	"io"
)

// Listener emits container events retrieved from a container runtime.
//
// Listen queues container events on the provided channel.
// Close stops the listener and cleans up.
type Listener interface {
	Listen(chan<- *container.Event)
	io.Closer
}
