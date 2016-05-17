package beacon

import (
	"github.com/pkg/errors"
	"sync"
)

// New creates a Beacon which receieves events from the `runtime` and uses
// `routes` to queue them into appropriate backends. New does not start the
// Beacon.
func New(runtime Runtime, routes []Route) (Beacon, error) {
	if runtime == nil {
		return nil, errors.New("runtime cannot be nil")
	}
	if len(routes) == 0 {
		return nil, errors.New("routes cannot be empty")
	}
	routesCp := make([]Route, len(routes))
	copy(routesCp, routes)
	return &beacon{
		runtime:    runtime,
		routes:     routesCp,
		containers: map[string]*Container{},
		lock:       &sync.Mutex{},
	}, nil
}

// Beacon processes events from a runtime into one or more backends. It
// maintains state for containers it sees and provides methods for interacting
// with that state.
type Beacon interface {
	// Run retrieves events from a runtime and routes them to the appropriate
	// backends. Run also maintains internal state for the Containers method.
	//
	// Run returns an error immediately if the runtime's EmitEvents method
	// fails. It will otherwise block until the runtime channel is closed.
	//
	// The runtime channel can be closed by calling Close either on Beacon or
	// by calling Close on the runtime directly. Beacon's Close method simply
	// wraps the runtime's Close method.
	//
	// Run will drain the channel of events before calling Close on each of the
	// backends. It then exits. An error is returned during if a failure occurs
	// in any of these steps. This error should be considered fatal as there is
	// no way to know the state of each of the involved components.
	Run() error

	// Close the beacon. Stops a running beacon and cleans up any resources
	// attached to it.
	Close() error

	// Containers retrieves the list of containers that Beacon has discovered.
	// An optional filter may be provided in order to limit the containers
	// returned.
	Containers(filter Filter) []*Container
}

// beacon is the standard Beacon implementation.
type beacon struct {
	runtime    Runtime
	routes     []Route
	containers map[string]*Container
	lock       *sync.Mutex
}

// Run the beacon.
func (b *beacon) Run() error {
	defer func() {
		for _, route := range b.routes {
			route.Close()
		}
	}()

	events, err := b.runtime.EmitEvents()
	if err != nil {
		return errors.Wrap(err, "failed to start runtime")
	}

	for {
		event, ok := <-events
		if !ok {
			break
		}
		if err := b.handle(event); err != nil {
			Logger.Printf("unable to process event: %s\n", err)
		}
	}
	return nil
}

// handle an event
func (b *beacon) handle(event *Event) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	var backendEvent *Event
	switch event.Action {
	case Start, Update:
		oldContainer, exists := b.containers[event.Container.ID]
		if !exists {
			// container does not exist and needs to be started
			newContainer := CopyContainer(event.Container)
			b.containers[event.Container.ID] = newContainer
			backendEvent = &Event{
				Action:    Start,
				Container: newContainer,
			}
		} else if !event.Container.Equal(oldContainer) {
			// container exists and needs to be updated
			newContainer := CopyContainer(event.Container)
			b.containers[event.Container.ID] = newContainer
			backendEvent = &Event{
				Action:    Update,
				Container: newContainer,
			}
		} else {
			// no change to an existing container
			return nil
		}
	case Stop:
		if oldContainer, exists := b.containers[event.Container.ID]; exists {
			// container exists and needs to be stopped
			delete(b.containers, event.Container.ID)
			backendEvent = &Event{
				Action:    event.Action,
				Container: oldContainer,
			}
		} else {
			// container already stopped
			return nil
		}
	default:
		return errors.Errorf("invalid action %s on container %s", event.Action, event.Container.ID)
	}

	for _, route := range b.routes {
		if route.MatchContainer(backendEvent.Container) {
			if err := route.ProcessEvent(CopyEvent(backendEvent)); err != nil {
				Logger.Printf("discarding event %s for container %s: %s", event.Action, event.Container.ID, err)
			}
		}
	}
	return nil
}

// Containers returns containers matching the given filter.
func (b *beacon) Containers(filter Filter) []*Container {
	if filter == nil {
		filter = &allFilter{}
	}

	b.lock.Lock()
	defer b.lock.Unlock()
	containers := []*Container{}
	for _, container := range b.containers {
		if filter == nil || filter.MatchContainer(container) {
			containers = append(containers, CopyContainer(container))
		}
	}
	return containers
}

// Close the beacon.
func (b *beacon) Close() error {
	b.runtime.Close()
	return nil
}
