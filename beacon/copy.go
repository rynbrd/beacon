package beacon

// CopyLabels returns a copy of the given labels.
func CopyLabels(labels map[string]string) map[string]string {
	labelsCp := make(map[string]string, len(labels))
	for name, value := range labels {
		labelsCp[name] = value
	}
	return labelsCp
}

// CopyBindings returns a copy of the given bindings.
func CopyBindings(bindings []*Binding) []*Binding {
	bindingsCp := make([]*Binding, len(bindings))
	for n, binding := range bindings {
		bindingCp := *binding
		bindingsCp[n] = &bindingCp
	}
	return bindingsCp
}

// CopyContainer returns a copy of the given container.
func CopyContainer(container *Container) *Container {
	return &Container{
		ID:       container.ID,
		Service:  container.Service,
		Labels:   CopyLabels(container.Labels),
		Hostname: container.Hostname,
		Bindings: CopyBindings(container.Bindings),
	}
}

// CopyEvent returns a copy of the given event.
func CopyEvent(event *Event) *Event {
	return &Event{
		Action:    event.Action,
		Container: CopyContainer(event.Container),
	}
}
