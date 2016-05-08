package beacon

import (
	"github.com/BlueDragonX/beacon/container"
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
	// The hostname to use when announcing mapped ports.
	Hostname string
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

	services map[serviceKey]*container.Address
	stopped  chan struct{}
}

// Run processes container events into services until Close is called.
func (b *Beacon) Run() {
	if len(b.Listeners) == 0 {
		logger.Fatal("no container listeners provided")
	}
	b.services = make(map[serviceKey]*container.Address)
	b.stopped = make(chan struct{})
	wg := &sync.WaitGroup{}
	events := make(chan *container.Event, 10)

	// start container listeners
	wg.Add(len(b.Listeners))
	for _, listener := range b.Listeners {
		go func(listener Listener) {
			defer wg.Done()
			listener.Listen(events)
		}(listener)
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
			logger.Print(err.Error())
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

func (b *Beacon) process(events <-chan *container.Event) {
	ticker := time.NewTicker(b.Heartbeat)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Action == container.Add {
				b.add(event.Container)
			} else if event.Action == container.Remove {
				b.remove(event.Container)
			}
		case <-ticker.C:
			b.heartbeat()
		}
	}
}

// add a container's services
func (b *Beacon) add(cntr *container.Container) {
	cntrServices, err := b.containerServices(cntr)
	if err != nil {
		logger.Print(err.Error())
		return
	}

	if len(cntrServices) == 0 {
		logger.Printf("container %s published no services", cntr.ID)
		return
	}

	ttl := b.Heartbeat + b.TTL
	for _, cntrService := range cntrServices {
		addr, err := b.containerAddress(cntr, cntrService.Port)
		if err != nil {
			logger.Print(err.Error())
			return
		}

		key := serviceKey{cntrService.Name, cntr.ID}
		if oldAddr, has := b.services[key]; !has || !addr.Equal(oldAddr) {
			b.services[key] = addr
			if err := b.Discovery.Announce(key.Name, key.Container, addr, ttl); err != nil {
				logger.Print(err.Error())
			}
		}
	}
}

// remove a container's services
func (b *Beacon) remove(cntr *container.Container) {
	cntrServices, err := b.containerServices(cntr)
	if err != nil {
		logger.Print(err.Error())
		return
	}

	for _, cntrService := range cntrServices {
		key := serviceKey{cntrService.Name, cntr.ID}
		if _, has := b.services[key]; has {
			delete(b.services, key)
			if err := b.Discovery.Shutdown(key.Name, key.Container); err != nil {
				logger.Print(err.Error())
			}
		}
	}
}

// heartbeat all services
func (b *Beacon) heartbeat() {
	logger.Print("heartbeat started")
	ttl := b.Heartbeat + b.TTL
	for svc, addr := range b.services {
		if err := b.Discovery.Announce(svc.Name, svc.Container, addr, ttl); err != nil {
			logger.Print(err.Error())
		}
	}
	logger.Print("heartbeat complete")
}

// destroy all container services
func (b *Beacon) destroy() {
	for key := range b.services {
		if _, has := b.services[key]; has {
			delete(b.services, key)
			if err := b.Discovery.Shutdown(key.Name, key.Container); err != nil {
				logger.Print(err.Error())
			}
		}
	}
}

// containerServices retrieves the services defined on a container.
func (b *Beacon) containerServices(cntr *container.Container) ([]*container.Service, error) {
	svcs := []*container.Service{}
	envVal := cntr.Env(b.EnvVar)
	for _, svcStr := range strings.Split(envVal, ",") {
		svcStr := strings.ToLower(svcStr)
		if svcStr == "" {
			continue
		}
		if svc, err := container.ParseService(svcStr); err == nil {
			svcs = append(svcs, svc)
		} else {
			return nil, err
		}
	}
	return svcs, nil
}

// containerAddress retrieves the address for a port on the container. The
// value of `beacon.hostname` will be used if the port is mapped but does not
// return a valid hostname. A valid hostname is any value which is not an empty
// string or "0.0.0.0".
func (b *Beacon) containerAddress(cntr *container.Container, port *container.Port) (*container.Address, error) {
	if addr, err := cntr.Mapping(port); err == nil {
		if addr.Hostname == "" || addr.Hostname == "0.0.0.0" {
			addr.Hostname = b.Hostname
		}
		return addr, nil
	} else if err == container.PortNotMapped {
		return &container.Address{
			Hostname: cntr.Hostname,
			Port:     port,
		}, nil
	} else {
		return nil, err
	}
}
