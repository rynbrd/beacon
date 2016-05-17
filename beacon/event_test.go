package beacon_test

import (
	beacon "."
	"reflect"
	"testing"
)

func TestEventCopy(t *testing.T) {
	event := &beacon.Event{
		Action: beacon.Start,
		Container: &beacon.Container{
			ID:      "123456",
			Service: "example",
			Labels: map[string]string{
				"a": "aye",
				"b": "bee",
			},
			Bindings: []*beacon.Binding{
				{
					HostIP:        "127.0.0.1",
					HostPort:      56291,
					ContainerPort: 80,
					Protocol:      beacon.TCP,
				},
				{
					HostIP:        "127.0.0.1",
					HostPort:      56292,
					ContainerPort: 443,
					Protocol:      beacon.TCP,
				},
			},
		},
	}
	newEvent := event.Copy()

	if !reflect.DeepEqual(event, newEvent) {
		t.Errorf("event copy differs from original: %+v != %+v", event, newEvent)
	}
}
