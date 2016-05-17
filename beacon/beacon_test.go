package beacon_test

import (
	beacon "."
	"github.com/pkg/errors"
	"sync"
	"testing"
	"time"
)

// MockRuntime emulates a real Runtime implementation.
type MockRuntime struct {
	Events chan *beacon.Event
}

// EmitEvents returns the MockRuntime.Events channel.
func (r *MockRuntime) EmitEvents() (<-chan *beacon.Event, error) {
	return r.Events, nil
}

func NewRuntime() *MockRuntime {
	return &MockRuntime{Events: make(chan *beacon.Event)}
}

// Close is a noop.
func (r *MockRuntime) Close() error {
	close(r.Events)
	return nil
}

// MockBackend emulates a real Backend implementation.
type MockBackend struct {
	Events chan *beacon.Event
}

func NewBackend() *MockBackend {
	return &MockBackend{Events: make(chan *beacon.Event)}
}

// ProcessEvent adds the event to the backend.
func (b *MockBackend) ProcessEvent(event *beacon.Event) error {
	b.Events <- event
	return nil
}

// Events waits up to `timeout` for `n` events and returns them. Return an
// error if the timeout occurs before recieving the requested number of events.
func (b *MockBackend) WaitForEvents(n int, timeout time.Duration) ([]*beacon.Event, error) {
	wait := time.After(timeout)
	events := make([]*beacon.Event, 0, n)
	var err error

Loop:
	for i := 0; i < n; i++ {
		select {
		case event := <-b.Events:
			events = append(events, event)
		case <-wait:
			err = errors.New("timed out waiting for events")
			break Loop
		}
	}
	return events, err
}

// Close is a noop.
func (b *MockBackend) Close() error {
	return nil
}

func ContainersEqual(a *beacon.Container, b *beacon.Container) error {
	if a.ID != b.ID {
		return errors.Errorf("container.ID inequal: %s != %s", a.ID, b.ID)
	}
	if a.Service != b.Service {
		return errors.Errorf("container.Service inequal: %s != %s", a.Service, b.Service)
	}
	if len(a.Labels) != len(b.Labels) {
		return errors.Errorf("container.Labels inequal length: %d != %d", len(a.Labels), len(b.Labels))
	}
	for k, v1 := range a.Labels {
		if v2, ok := b.Labels[k]; !ok || v1 != v2 {
			return errors.Errorf("container.Labels[%s] inequal: %s != %s", k, v1, v2)
		}
	}
	if len(a.Bindings) != len(b.Bindings) {
		return errors.Errorf("container.bindings have length: %d != %d", len(a.Bindings), len(b.Bindings))
	}
	for n, b1 := range a.Bindings {
		b2 := b.Bindings[n]
		if b1.HostPort != b2.HostPort || b1.ContainerPort != b2.ContainerPort || b1.Protocol != b2.Protocol {
			return errors.Errorf("container.Bindings[%d] inequal: %+v != %+v", n, b1, b2)
		}
	}
	return nil
}

func ContainerSetsEqual(a []*beacon.Container, b []*beacon.Container) error {
	if len(a) != len(b) {
		return errors.Errorf("container sets inequal length: %d != %d", len(a), len(b))
	}

	newSet := func(arr []*beacon.Container) map[string]*beacon.Container {
		set := make(map[string]*beacon.Container, len(arr))
		for _, cntr := range arr {
			set[cntr.ID] = cntr
		}
		return set
	}

	aSet := newSet(a)
	bSet := newSet(b)

	for id, c1 := range aSet {
		if c2, ok := bSet[id]; !ok {
			return errors.Errorf("container[%s] not in both sets", id)
		} else if err := ContainersEqual(c1, c2); err != nil {
			return errors.Wrapf(err, "container[%s] inequal", id)
		}
	}
	return nil
}

func EventsEqual(a *beacon.Event, b *beacon.Event) error {
	if a.Action != b.Action {
		return errors.Errorf("event.Action inequal: %s != %s", a.Action, b.Action)
	}
	if err := ContainersEqual(a.Container, b.Container); err != nil {
		return errors.Wrap(err, "event.Container inequal")
	}
	return nil
}

func EventArraysEqual(a []*beacon.Event, b []*beacon.Event) error {
	if len(a) != len(b) {
		return errors.Errorf("event arrays have inequal length: %d != %d", len(a), len(b))
	}
	for n := range a {
		if err := EventsEqual(a[n], b[n]); err != nil {
			return errors.Wrapf(err, "events[%d] inequal", n)
		}
	}
	return nil
}

// Test that beacon.New raises an error when provided invalid inputs.
func TestBeaconNewError(t *testing.T) {
	if _, err := beacon.New(nil, []beacon.Route{}); err == nil {
		t.Error("expected error for empty arguments")
	}

	if _, err := beacon.New(NewRuntime(), []beacon.Route{}); err == nil {
		t.Error("expected error for empty route list")
	}

	filter := beacon.NewFilter(nil)
	route := beacon.NewRoute(filter, NewBackend())
	if _, err := beacon.New(nil, []beacon.Route{route}); err == nil {
		t.Error("expected error for nil runtime")
	}
}

func TestBeaconRunOneBackend(t *testing.T) {
	runtime := NewRuntime()
	backend := NewBackend()
	route := beacon.NewRoute(nil, backend)
	bcn, err := beacon.New(runtime, []beacon.Route{route})
	if err != nil {
		t.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bcn.Run(); err != nil {
			t.Fatal(err)
		}
	}()

	queueEvents := []*beacon.Event{
		{
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
		},
		{
			Action: beacon.Update,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "eh",
					"c": "see",
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
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID: "123456",
			},
		},
	}
	wantEvents := []*beacon.Event{
		{
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
		},
		{
			Action: beacon.Update,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "eh",
					"c": "see",
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
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "eh",
					"c": "see",
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
		},
	}

	go func() {
		for _, event := range queueEvents {
			runtime.Events <- event
		}
	}()

	haveEvents, err := backend.WaitForEvents(3, 5*time.Second)
	if err != nil {
		t.Error(err)
	}
	if err := EventArraysEqual(haveEvents, wantEvents); err != nil {
		t.Error(err)
	}

	if err := bcn.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBeaconRunTwoBackends(t *testing.T) {
	runtime := NewRuntime()
	backend1 := NewBackend()
	backend2 := NewBackend()
	routes := []beacon.Route{
		beacon.NewRoute(nil, backend1),
		beacon.NewRoute(nil, backend2),
	}
	bcn, err := beacon.New(runtime, routes)
	if err != nil {
		t.Fatal(err)
	}

	runWait := &sync.WaitGroup{}
	runWait.Add(1)
	go func() {
		defer runWait.Done()
		if err := bcn.Run(); err != nil {
			t.Fatal(err)
		}
	}()

	queueEvents := []*beacon.Event{
		{
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
		},
		{
			Action: beacon.Update,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "eh",
					"c": "see",
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
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID: "123456",
			},
		},
	}
	wantEvents := []*beacon.Event{
		{
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
		},
		{
			Action: beacon.Update,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "eh",
					"c": "see",
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
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "eh",
					"c": "see",
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
		},
	}

	go func() {
		for _, event := range queueEvents {
			runtime.Events <- event
		}
	}()

	eventWait := &sync.WaitGroup{}
	checkEvents := func(backend *MockBackend, wantEvents []*beacon.Event) {
		defer eventWait.Done()
		haveEvents, err := backend.WaitForEvents(3, 5*time.Second)
		if err != nil {
			t.Error(err)
		}
		if err := EventArraysEqual(haveEvents, wantEvents); err != nil {
			t.Error(err)
		}
	}

	eventWait.Add(2)
	go checkEvents(backend1, wantEvents)
	go checkEvents(backend2, wantEvents)
	eventWait.Wait()

	if err := bcn.Close(); err != nil {
		t.Fatal(err)
	}
	runWait.Wait()
}

func TestBeaconRunFilterBackends(t *testing.T) {
	runtime := NewRuntime()
	backend1 := NewBackend()
	backend2 := NewBackend()

	filter1, err := beacon.ParseFilter("color=blue")
	if err != nil {
		t.Fatal(err)
	}

	filter2, err := beacon.ParseFilter("color=green")
	if err != nil {
		t.Fatal(err)
	}

	routes := []beacon.Route{
		beacon.NewRoute(filter1, backend1),
		beacon.NewRoute(filter2, backend2),
	}

	bcn, err := beacon.New(runtime, routes)
	if err != nil {
		t.Fatal(err)
	}

	runWait := &sync.WaitGroup{}
	runWait.Add(1)
	go func() {
		defer runWait.Done()
		if err := bcn.Run(); err != nil {
			t.Fatal(err)
		}
	}()

	container1 := &beacon.Container{
		ID:      "1",
		Service: "example",
		Labels: map[string]string{
			"color": "blue",
		},
		Bindings: []*beacon.Binding{},
	}

	container2 := &beacon.Container{
		ID:      "2",
		Service: "example",
		Labels: map[string]string{
			"color": "green",
		},
		Bindings: []*beacon.Binding{},
	}

	queueEvents := []*beacon.Event{
		{
			Action:    beacon.Start,
			Container: container1,
		},
		{
			Action:    beacon.Start,
			Container: container2,
		},
	}
	wantBlueEvents := []*beacon.Event{
		{
			Action:    beacon.Start,
			Container: container1,
		},
	}
	wantGreenEvents := []*beacon.Event{
		{
			Action:    beacon.Start,
			Container: container2,
		},
	}

	go func() {
		for _, event := range queueEvents {
			runtime.Events <- event
		}
	}()

	eventWait := &sync.WaitGroup{}
	checkEvents := func(backend *MockBackend, wantEvents []*beacon.Event) {
		defer eventWait.Done()
		haveEvents, err := backend.WaitForEvents(1, 5*time.Second)
		if err != nil {
			t.Error(err)
		}
		if err := EventArraysEqual(haveEvents, wantEvents); err != nil {
			t.Error(err)
		}
	}

	eventWait.Add(2)
	go checkEvents(backend1, wantBlueEvents)
	go checkEvents(backend2, wantGreenEvents)
	eventWait.Wait()

	if err := bcn.Close(); err != nil {
		t.Fatal(err)
	}
	runWait.Wait()
}

func TestBeaconContainers(t *testing.T) {
	runtime := NewRuntime()
	backend := NewBackend()
	routes := []beacon.Route{
		beacon.NewRoute(nil, backend),
	}

	bcn, err := beacon.New(runtime, routes)
	if err != nil {
		t.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bcn.Run(); err != nil {
			t.Fatal(err)
		}
	}()

	container1 := &beacon.Container{
		ID:      "1",
		Service: "example",
		Labels: map[string]string{
			"color": "red",
		},
		Bindings: []*beacon.Binding{},
	}

	container2 := &beacon.Container{
		ID:      "2",
		Service: "example",
		Labels: map[string]string{
			"color": "green",
		},
		Bindings: []*beacon.Binding{},
	}

	container3 := &beacon.Container{
		ID:      "3",
		Service: "example",
		Labels: map[string]string{
			"color": "blue",
		},
		Bindings: []*beacon.Binding{},
	}

	go func() {
		runtime.Events <- &beacon.Event{
			Action:    beacon.Start,
			Container: container1,
		}
		runtime.Events <- &beacon.Event{
			Action:    beacon.Start,
			Container: container2,
		}
		runtime.Events <- &beacon.Event{
			Action:    beacon.Start,
			Container: container3,
		}
	}()

	if _, err := backend.WaitForEvents(3, 5*time.Second); err != nil {
		t.Fatal(err)
	}

	haveContainers := bcn.Containers(nil)
	wantContainers := []*beacon.Container{
		container1,
		container2,
		container3,
	}
	if err := ContainerSetsEqual(haveContainers, wantContainers); err != nil {
		t.Error(err)
	}

	filter, err := beacon.ParseFilter("color=blue")
	if err != nil {
		t.Fatal(err)
	}
	haveContainers = bcn.Containers(filter)
	wantContainers = []*beacon.Container{
		container3,
	}
	if err := ContainerSetsEqual(haveContainers, wantContainers); err != nil {
		t.Error(err)
	}

	if err := bcn.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBeaconRunOneStopTwice(t *testing.T) {
	runtime := NewRuntime()
	backend := NewBackend()
	route := beacon.NewRoute(nil, backend)
	bcn, err := beacon.New(runtime, []beacon.Route{route})
	if err != nil {
		t.Fatal(err)
	}

	runWait := &sync.WaitGroup{}
	runWait.Add(1)
	go func() {
		defer runWait.Done()
		if err := bcn.Run(); err != nil {
			t.Fatal(err)
		}
	}()

	queueEvents := []*beacon.Event{
		{
			Action: beacon.Start,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "aye",
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
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID: "123456",
			},
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID: "123456",
			},
		},
	}
	wantEvents := []*beacon.Event{
		{
			Action: beacon.Start,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "aye",
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
		},
		{
			Action: beacon.Stop,
			Container: &beacon.Container{
				ID:      "123456",
				Service: "example",
				Labels: map[string]string{
					"a": "aye",
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
		},
	}

	eventWait := &sync.WaitGroup{}
	eventWait.Add(1)
	go func() {
		defer eventWait.Done()
		for _, event := range queueEvents {
			runtime.Events <- event
		}
	}()

	haveEvents, err := backend.WaitForEvents(2, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := EventArraysEqual(haveEvents, wantEvents); err != nil {
		t.Fatal(err)
	}
	eventWait.Wait()

	if err := bcn.Close(); err != nil {
		t.Fatal(err)
	}
	runWait.Wait()
}
