package beacon

// Protocol is the IP protocol of a container port.
type Protocol string

// Available protocol values.
const (
	TCP Protocol = "tcp" // The port speaks TCP.
	UDP          = "udp" // The port speaks UDP.
)

// Binding represents a bound container port on a host.
type Binding struct {
	HostPort      int      // The port on the host that maps to the container's port.
	ContainerPort int      // The port the container is listening on.
	Protocol      Protocol // The protocol the port is configured for.
}
