package beacon

// Container represents a single container belonging to a service. All fields
// are considered to be immutable with exceptions for the state fields.
type Container struct {
	// The globally unique identifier of the container. This is typically set
	// by the underlying container implementation.
	ID string

	// The name of the service to which the container belongs. Determined by
	// the event listener.
	Service string

	// The label metadata attached to the container.
	Labels map[string]string

	// Network port bindings.
	Bindings []*Binding
}

// Equal returns true if this container is equal to another.
func (a *Container) Equal(b *Container) bool {
	if a == nil && b == nil {
		return true
	} else if a == nil || b == nil {
		return false
	}
	if a.ID != b.ID || a.Service != b.Service || len(a.Labels) != len(b.Labels) || len(a.Bindings) != len(b.Bindings) {
		return false
	}
	for name, val1 := range a.Labels {
		if val2, ok := b.Labels[name]; !ok || val1 != val2 {
			return false
		}
	}
	for n, binding1 := range a.Bindings {
		if !binding1.Equal(b.Bindings[n]) {
			return false
		}
	}
	return true
}
