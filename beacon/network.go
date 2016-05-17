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

// Equal returns true if this binding is equal to another.
func (a *Binding) Equal(b *Binding) bool {
	if a == nil && b == nil {
		return true
	} else if a == nil || b == nil {
		return false
	}
	return a.HostIP == b.HostIP && a.HostPort == b.HostPort && a.ContainerPort == b.ContainerPort && a.Protocol == b.Protocol
}
