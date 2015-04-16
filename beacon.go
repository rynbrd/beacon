package main

import (
	"strings"
	"sync"
	"time"
)

// serviceKey is used to uniquely identify a service address.
type serviceKey struct {
	Name      string
	Container string
}

// Beacon listens for containers coming online and announces their published
// service into etcd. When a container goes offline those published services
// are removed.
type Beacon struct {
	// The interval between heartbeats.
	Heartbeat time.Duration
	// How long to keep a service available after a heartbeat is missed.
	TTL time.Duration
	// The container environment variable containing the hosted services.
	EnvVar string
	// The listeners to recieve container events from.
	Listeners []Listener
	// The discovery backend to announce services to.
	Discovery Discovery

	services map[serviceKey]*Address
	stopped  chan struct{}
}

// Run processes container events into services until Close is called.
func (b *Beacon) Run() {
	if len(b.Listeners) == 0 {
		logger.Fatal("no container listeners provided")
	}
	b.services = make(map[serviceKey]*Address)
	b.stopped = make(chan struct{})
	wg := &sync.WaitGroup{}
	events := make(chan *ContainerEvent, 10)

	// start container listeners
	wg.Add(len(b.Listeners))
	for _, listener := range b.Listeners {
		go func() {
			defer wg.Done()
			listener.Listen(events)
		}()
	}

	// process container events
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.process(events)
	}()

	// wait for completion
	<-b.stopped

	// stop container listeners
	for _, listener := range b.Listeners {
		if err := listener.Close(); err != nil {
			logger.Error(err.Error())
		}
	}

	// clean up and wait for goroutines
	close(events)
	b.destroy()
	wg.Wait()
}

// Close stops processing container events.
func (b *Beacon) Close() error {
	close(b.stopped)
	return nil
}

func (b *Beacon) process(events <-chan *ContainerEvent) {
	ticker := time.NewTicker(b.Heartbeat)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Action == ContainerAdd {
				b.add(event.Container)
			} else if event.Action == ContainerRemove {
				b.remove(event.Container)
			}
		case <-ticker.C:
			b.heartbeat()
		}
	}
}

// add a container's services
func (b *Beacon) add(container *Container) {
	cntrServices, err := b.containerServices(container)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if len(cntrServices) == 0 {
		logger.Debugf("container %s published no services", container.ID)
		return
	}

	ttl := b.Heartbeat + b.TTL
	for _, cntrService := range cntrServices {
		addr, err := b.containerAddress(container, cntrService.Port)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		key := serviceKey{cntrService.Name, container.ID}
		if oldAddr, has := b.services[key]; !has || !addr.Equal(oldAddr) {
			b.services[key] = addr
			if err := b.Discovery.Announce(key.Name, key.Container, addr, ttl); err != nil {
				logger.Error(err.Error())
			}
		}
	}
}

// remove a container's services
func (b *Beacon) remove(container *Container) {
	cntrServices, err := b.containerServices(container)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	for _, cntrService := range cntrServices {
		key := serviceKey{cntrService.Name, container.ID}
		if _, has := b.services[key]; has {
			delete(b.services, key)
			if err := b.Discovery.Shutdown(key.Name, key.Container); err != nil {
				logger.Error(err.Error())
			}
		}
	}
}

// heartbeat all services
func (b *Beacon) heartbeat() {
	logger.Debug("heartbeat started")
	ttl := b.Heartbeat + b.TTL
	for svc, addr := range b.services {
		if err := b.Discovery.Announce(svc.Name, svc.Container, addr, ttl); err != nil {
			logger.Error(err.Error())
		}
	}
	logger.Debug("heartbeat complete")
}

// destroy all container services
func (b *Beacon) destroy() {
	for key := range b.services {
		if _, has := b.services[key]; has {
			delete(b.services, key)
			if err := b.Discovery.Shutdown(key.Name, key.Container); err != nil {
				logger.Error(err.Error())
			}
		}
	}
}

// containerServices retrieves the services defined on a container.
func (b *Beacon) containerServices(container *Container) ([]*ContainerService, error) {
	svcs := []*ContainerService{}
	envVal := container.Env(b.EnvVar)
	for _, svcStr := range strings.Split(envVal, ",") {
		if svc, err := ParseContainerService(svcStr); err == nil {
			svcs = append(svcs, svc)
		} else {
			return nil, err
		}
	}
	return svcs, nil
}

// containerAddress retrieves the address for a port on the container.
func (b *Beacon) containerAddress(container *Container, port *Port) (*Address, error) {
	if addr, err := container.Mapping(port); err == nil {
		return addr, nil
	} else if err == PortNotMapped {
		return &Address{container.Hostname, port}, nil
	} else {
		return nil, err
	}
}
