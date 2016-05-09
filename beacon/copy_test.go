package beacon_test

import (
	beacon "."
	"reflect"
	"testing"
)

func TestCopyLabels(t *testing.T) {
	labels := map[string]string{
		"a": "aye",
		"b": "bee",
	}
	newLabels := beacon.CopyLabels(labels)

	if !reflect.DeepEqual(labels, newLabels) {
		t.Errorf("label copy differs from original: %+v != %+v", labels, newLabels)
	}

	newValue := "eh"
	labels["a"] = newValue
	if newLabels["a"] == newValue {
		t.Error("label copy points to same memory space")
	}
}

func TestCopyBindings(t *testing.T) {
	bindings := []*beacon.Binding{
		{
			HostPort:      56291,
			ContainerPort: 80,
			Protocol:      beacon.TCP,
		},
		{
			HostPort:      56292,
			ContainerPort: 443,
			Protocol:      beacon.TCP,
		},
	}
	newBindings := beacon.CopyBindings(bindings)

	if !reflect.DeepEqual(bindings, newBindings) {
		t.Errorf("binding copy differs from original: %+v != %+v", bindings, newBindings)
	}

	newPort := 54293
	bindings[1].HostPort = newPort
	if newBindings[1].HostPort == newPort {
		t.Error("binding copy points to same memory space")
	}
}

func TestCopyContainer(t *testing.T) {
	container := &beacon.Container{
		ID:      "123456",
		Service: "example",
		Labels: map[string]string{
			"a": "aye",
			"b": "bee",
		},
		Hostname: "localhost",
		Bindings: []*beacon.Binding{
			{
				HostPort:      56291,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			{
				HostPort:      56292,
				ContainerPort: 443,
				Protocol:      beacon.TCP,
			},
		},
	}
	newContainer := beacon.CopyContainer(container)

	if !reflect.DeepEqual(container, newContainer) {
		t.Errorf("container copy differs from original: %+v != %+v", container, newContainer)
	}

	newService := "www"
	container.Service = newService
	if newContainer.Service == newService {
		t.Error("container copy points to same memory space")
	}

	newLabel := "eh"
	container.Labels["a"] = newLabel
	if reflect.DeepEqual(container.Labels, newContainer.Labels) {
		t.Error("container.Labels copy points to same memory space")
	}

	newPort := 54293
	container.Bindings[1].HostPort = newPort
	if reflect.DeepEqual(container.Bindings, newContainer.Bindings) {
		t.Error("container.Bindings copy points to same memory space")
	}
}

func TestCopyEvent(t *testing.T) {
	event := &beacon.Event{
		Action: beacon.Start,
		Container: &beacon.Container{
			ID:      "123456",
			Service: "example",
			Labels: map[string]string{
				"a": "aye",
				"b": "bee",
			},
			Hostname: "localhost",
			Bindings: []*beacon.Binding{
				{
					HostPort:      56291,
					ContainerPort: 80,
					Protocol:      beacon.TCP,
				},
				{
					HostPort:      56292,
					ContainerPort: 443,
					Protocol:      beacon.TCP,
				},
			},
		},
	}
	newEvent := beacon.CopyEvent(event)

	if !reflect.DeepEqual(event, newEvent) {
		t.Errorf("event copy differs from original: %+v != %+v", event, newEvent)
	}
}
