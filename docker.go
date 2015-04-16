package main

import (
	"crypto/tls"
	"github.com/BlueDragonX/dockerclient"
	"strconv"
	"time"
)

// Docker provides container events from a Docker container runtime.
type Docker struct {
	client     *dockerclient.DockerClient
	interval   time.Duration
	containers map[string]*Container
	stopped    chan struct{}
}

// NewDocker creates a Docker object connected to `uri`. It will listen for
// events and poll after `interval` to ensure no events were missed. TLS may be
// enabled by providing a non-nil value to `tls`.
func NewDocker(uri string, interval time.Duration, tls *tls.Config) (*Docker, error) {
	if client, err := dockerclient.NewDockerClient(uri, tls); err == nil {
		docker := &Docker{
			client,
			interval,
			make(map[string]*Container),
			make(chan struct{}),
		}
		return docker, nil
	} else {
		return nil, err
	}
}

// Listen for container events and queue them into `events`.
func (docker *Docker) Listen(events chan<- *ContainerEvent) {
	logger.Debugf("docker listener started")

	// listen for events from docker
	clientEvents := make(chan *dockerclient.Event)
	docker.client.StartMonitorEvents(func(e *dockerclient.Event, args ...interface{}) {
		clientEvents <- e
	})

	// do an initial poll to load the current containers
	docker.poll(events)

	// process client events and poll periodically
	ticker := time.NewTicker(docker.interval)
	defer ticker.Stop()
Loop:
	for {
		select {
		case e := <-clientEvents:
			// process client events from monitor
			if e.Status == "start" || e.Status == "unpause" {
				docker.add(e.Id, events)
			} else if e.Status == "die" || e.Status == "kill" || e.Status == "pause" {
				docker.remove(e.Id, events)
			}
		case <-ticker.C:
			// poll for container list
			docker.poll(events)
		case <-docker.stopped:
			docker.client.StopAllMonitorEvents()
			break Loop
		}
	}
	logger.Debugf("docker listener stopped")
}

// Close stops listening for container events.
func (docker *Docker) Close() error {
	close(docker.stopped)
	return nil
}

func (docker *Docker) poll(events chan<- *ContainerEvent) {
	logger.Debugf("docker poll started")
	containers, err := docker.client.ListContainers(false)
	if err != nil {
		logger.Errorf("list containers failed: %s", err)
	}
	ids := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		ids[container.Id] = struct{}{}
		docker.add(container.Id, events)
	}
	for id := range docker.containers {
		if _, has := ids[id]; !has {
			docker.remove(id, events)
		}
	}
	logger.Debugf("docker poll complete")
}

// add emits an Add event for the container with the given id.
func (docker *Docker) add(id string, events chan<- *ContainerEvent) {
	if _, has := docker.containers[id]; has {
		return
	}
	if container := docker.get(id); container != nil {
		logger.Debugf("docker started container %s", id)
		docker.containers[id] = container
		events <- &ContainerEvent{ContainerAdd, container}
	}
}

// remove emits a Remove event for the container with the given id.
func (docker *Docker) remove(id string, events chan<- *ContainerEvent) {
	if container, has := docker.containers[id]; has {
		logger.Debugf("docker stopped container %s", id)
		delete(docker.containers, id)
		events <- &ContainerEvent{ContainerRemove, container}
	}
}

// get the container which has the given id. Logs an error and returns nil if not found.
func (docker *Docker) get(id string) *Container {
	errorFmt := "docker inspect failed on %s: %s"
	info, err := docker.client.InspectContainer(id)
	if err != nil {
		logger.Errorf(errorFmt, id, err)
		return nil
	}

	mappings := []*Mapping{}
	for port, bindings := range info.NetworkSettings.Ports {
		containerPort, err := ParsePort(port)
		if err != nil {
			logger.Errorf(errorFmt, id, err)
			return nil
		}
		for _, binding := range bindings {
			hostPort, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				logger.Errorf(errorFmt, id, err)
				return nil
			}
			hostAddress := &Address{binding.HostIp, &Port{hostPort, containerPort.Protocol}}
			mappings = append(mappings, &Mapping{hostAddress, containerPort})
		}
	}

	return &Container{
		ID:       info.Id,
		Environ:  info.Config.Env,
		Hostname: info.NetworkSettings.IpAddress,
		Mappings: mappings,
	}
}
