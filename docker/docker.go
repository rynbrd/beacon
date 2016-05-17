package docker

import (
	"github.com/BlueDragonX/beacon/beacon"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"strconv"
	"sync"
)

var (
	errContainerIgnored = errors.New("container ignored")
)

// New creates a Docker runtime from the provided configuration. The runtime
// listens for container events on the Docker `endpoint`.
//
// The hostIP is reported to Beacon as the IP address used to connect to
// discovered containers when they listen on all host addresses (0.0.0.0).
//
// The serviceLabel is used to look up the service name from the labels on the
// container. Containers without this label or with an empty serviceLabel are
// ignored.
//
// If stopOnClose is true then stop events will be queued for each running
// container when Close is called.
func New(endpoint string, hostIP, serviceLabel string, stopOnClose bool) (beacon.Runtime, error) {
	client, err := dockerclient.NewClient(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}
	if hostIP == "" {
		return nil, errors.Errorf("invalid hostIP %s", hostIP)
	}

	return &docker{
		client:       client,
		hostIP:       hostIP,
		serviceLabel: serviceLabel,
		stopOnClose:  stopOnClose,
		wg:           &sync.WaitGroup{},
		stop:         make(chan struct{}),
	}, nil
}

// docker implements a Beacon runtime for the Docker daemon.
type docker struct {
	client       *dockerclient.Client
	hostIP       string
	serviceLabel string
	stopOnClose  bool
	wg           *sync.WaitGroup
	stop         chan struct{}
}

// EmitEvents sends Docker events to Beacon.
func (d *docker) EmitEvents() (<-chan *beacon.Event, error) {
	dockerEvents := make(chan *dockerclient.APIEvents, 1)
	if err := d.client.AddEventListener(dockerEvents); err != nil {
		return nil, errors.Wrap(err, "failed to listen for docker events")
	}
	beaconEvents := make(chan *beacon.Event, 1)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(beaconEvents)

		running := map[string]struct{}{}

		defer func() {
			if d.stopOnClose {
				for id := range running {
					beaconEvents <- &beacon.Event{
						Action: beacon.Stop,
						Container: &beacon.Container{
							ID: id,
						},
					}
				}
			}
		}()

		sendStart := func(cntr *beacon.Container) bool {
			running[cntr.ID] = struct{}{}
			beaconEvent := &beacon.Event{
				Action:    beacon.Start,
				Container: cntr,
			}
			select {
			case beaconEvents <- beaconEvent:
			case <-d.stop:
				return false
			}
			return true
		}

		sendStop := func(id string) bool {
			if _, ok := running[id]; ok {
				delete(running, id)
			}
			beaconEvent := &beacon.Event{
				Action: beacon.Stop,
				Container: &beacon.Container{
					ID: id,
				},
			}
			select {
			case beaconEvents <- beaconEvent:
			case <-d.stop:
				return false
			}
			return true
		}

		// get and queue existing containers
		containers, err := d.listContainers()
		if err != nil {
			Logger.Print(err)
		}
		for _, container := range containers {
			if !sendStart(container) {
				return
			}
		}

		for {
			select {
			case dockerEvent, ok := <-dockerEvents:
				if !ok {
					return
				}
				switch dockerEvent.Action {
				case "start":
					if container, err := d.inspectContainer(dockerEvent.Actor.ID); err == nil {
						if !sendStart(container) {
							return
						}
					} else if err != errContainerIgnored {
						Logger.Printf("failed to inspect container %s: %s", dockerEvent.Actor.ID, err)
					}
				case "stop", "die":
					if !sendStop(dockerEvent.Actor.ID) {
						return
					}
				}
			case <-d.stop:
				return
			}
		}
	}()
	return beaconEvents, nil
}

func (d *docker) listContainers() ([]*beacon.Container, error) {
	opts := dockerclient.ListContainersOptions{
		Filters: map[string][]string{
			"status": {"running"},
		},
	}
	apiContainers, err := d.client.ListContainers(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list containers")
	}

	containers := make([]*beacon.Container, 0, len(apiContainers))
	for _, apiContainer := range apiContainers {
		container, err := d.inspectContainer(apiContainer.ID)
		if err == errContainerIgnored {
			continue
		} else if err != nil {
			return nil, errors.Wrap(err, "failed to list containers")
		}
		containers = append(containers, container)
	}
	return containers, nil
}

func (d *docker) inspectContainer(id string) (*beacon.Container, error) {
	dockerContainer, err := d.client.InspectContainer(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to inspect container %s", id)
	}

	service, ok := dockerContainer.Config.Labels[d.serviceLabel]
	if !ok {
		return nil, errContainerIgnored
	}

	bindings := make([]*beacon.Binding, 0, len(dockerContainer.HostConfig.PortBindings))
	for dockerPort, dockerBindings := range dockerContainer.NetworkSettings.Ports {
		containerPort, protocol, err := parsePort(string(dockerPort))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to inspect container %s", id)
		}
		for _, dockerBinding := range dockerBindings {
			hostIP := dockerBinding.HostIP
			if hostIP == "0.0.0.0" {
				hostIP = d.hostIP
			}
			hostPort, err := strconv.Atoi(dockerBinding.HostPort)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to inspect container %s", id)
			}
			bindings = append(bindings, &beacon.Binding{
				HostIP:        hostIP,
				HostPort:      hostPort,
				ContainerPort: containerPort,
				Protocol:      protocol,
			})
		}
	}

	return &beacon.Container{
		ID:       dockerContainer.ID,
		Service:  service,
		Labels:   dockerContainer.Config.Labels,
		Bindings: bindings,
	}, nil
}

// Close the connection to Docker and stop emiting events.
func (d *docker) Close() error {
	close(d.stop)
	d.wg.Wait()
	return nil
}
