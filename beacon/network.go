package beacon

// Protocol is the IP protocol of a container port.
type Protocol string

// Available protocol values.
const (
	TCP Protocol = "tcp" // The port speaks TCP.
	UDP          = "udp" // The port speaks UDP.
)

// Binding represents a bound container port on a host. This contains two
// logical components: the IP and port where an external service can connect to
// the container; and the internal port that the container itself is listening
// on. The protocol is, of course, the same for both the host port and
// container port.
type Binding struct {
	HostIP        string   // The IP on the host where the host port is accessible.
	HostPort      int      // The port on the host that maps to the container's port.
	ContainerPort int      // The port the container is listening on.
	Protocol      Protocol // The protocol the port is configured for.
}

// Equal returns true if this Binding is equal to another.
func (b *Binding) Equal(c *Binding) bool {
	if b == nil && c == nil {
		return true
	} else if b == nil || c == nil {
		return false
	}
	return b.HostIP == c.HostIP && b.HostPort == c.HostPort && b.ContainerPort == c.ContainerPort && b.Protocol == c.Protocol
}

// Copy allocates a copy of the Binding.
func (b *Binding) Copy() *Binding {
	if b == nil {
		return nil
	}
	cp := *b
	return &cp
}
