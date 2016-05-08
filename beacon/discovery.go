package beacon

import (
	"github.com/BlueDragonX/beacon/container"
	"io"
	"time"
)

// Discovery is a data store to which service announcements are made.
//
// Announce informs the backend that a service is being hosted by a container
// at the provided address. The backend should expire the address after TTL has
// elapsed.
//
// Shutdown indicates to the backend that the named service is no longer
// being provided by a container. This enables a service to be removed
// immediately instead of waiting for a TTL to expire.
type Discovery interface {
	Announce(name, container string, address *container.Address, ttl time.Duration) error
	Shutdown(name, container string) error
	io.Closer
}
