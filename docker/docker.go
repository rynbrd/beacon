package docker

import (
	"github.com/BlueDragonX/beacon/beacon"
	"github.com/fsouza/go-dockerclient"
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
func New(endpoint string, hostIP, serviceLabel string) (*Docker, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}
	if hostIP == "" {
		return nil, errors.Errorf("invalid hostIP %s", hostIP)
	}

	return &Docker{
		client:       client,
		hostIP:       hostIP,
		serviceLabel: serviceLabel,
		wg:           &sync.WaitGroup{},
		stop:         make(chan struct{}),
	}, nil
}

// Docker implements a Beacon runtime for the Docker daemon.
type Docker struct {
	client       *docker.Client
	hostIP       string
	serviceLabel string
	wg           *sync.WaitGroup
	stop         chan struct{}
}

// EmitEvents sends Docker events to Beacon.
func (d *Docker) EmitEvents() (<-chan *beacon.Event, error) {
	dockerEvents := make(chan *docker.APIEvents, 1)
	if err := d.client.AddEventListener(dockerEvents); err != nil {
		return nil, errors.Wrap(err, "failed to listen for docker events")
	}
	beaconEvents := make(chan *beacon.Event, 1)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(beaconEvents)

		// get and queue existing containers
		containers, err := d.listContainers()
		if err != nil {
			Logger.Print(err)
		}
		for _, container := range containers {
			beaconEvent := &beacon.Event{
				Action:    beacon.Start,
				Container: container,
			}
			select {
			case beaconEvents <- beaconEvent:
			case <-d.stop:
				return
			}
		}

		for {
			select {
			case dockerEvent, ok := <-dockerEvents:
				if !ok {
					break
				}
				switch dockerEvent.Action {
				case "start":
					if container, err := d.inspectContainer(dockerEvent.Actor.ID); err == nil {
						beaconEvent := &beacon.Event{
							Action:    beacon.Start,
							Container: container,
						}
						select {
						case beaconEvents <- beaconEvent:
						case <-d.stop:
							return
						}
					} else if err != errContainerIgnored {
						Logger.Printf("failed to inspect container %s: %s", dockerEvent.Actor.ID, err)
					}
				case "stop", "die":
					beaconEvent := &beacon.Event{
						Action: beacon.Stop,
						Container: &beacon.Container{
							ID: dockerEvent.Actor.ID,
						},
					}
					select {
					case beaconEvents <- beaconEvent:
					case <-d.stop:
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

func (d *Docker) listContainers() ([]*beacon.Container, error) {
	opts := docker.ListContainersOptions{
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

func (d *Docker) inspectContainer(id string) (*beacon.Container, error) {
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
func (d *Docker) Close() error {
	close(d.stop)
	d.wg.Wait()
	return nil
}
