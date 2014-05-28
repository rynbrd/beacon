package main

import (
	"errors"
	"github.com/samalba/dockerclient"
	"strings"
	"time"
)

// Monitor docker for service changes and emit events.
type ServiceMonitor struct {
	client       *dockerclient.DockerClient
	hostname     string
	configVar    string
	state        int32
	containers   map[string]bool
	services     map[string]*Service
	pollInterval time.Duration
	stop         chan bool
}

// Create a new service monitor listening on the given URL. Look for service
// config in the Docker environment variable names configVar.
func NewServiceMonitor(url, hostname, configVar string, pollInterval time.Duration) (mon *ServiceMonitor, err error) {
	mon = &ServiceMonitor{}
	mon.client, err = dockerclient.NewDockerClient(url)
	mon.hostname = hostname
	mon.configVar = configVar
	mon.state = Stopped
	mon.pollInterval = pollInterval
	mon.stop = make(chan bool)
	return
}

func (mon *ServiceMonitor) addContainer(serviceEvents chan ServiceEvent, containerId string) (err error) {
	var containerInfo *dockerclient.ContainerInfo
	if containerInfo, err = mon.client.InspectContainer(containerId); err != nil {
		return err
	}

	configEnv := ""
	for _, envVar := range containerInfo.Config.Env {
		if envName, envValue := parseEnv(envVar); envName == mon.configVar {
			configEnv = envValue
		}
	}

	configValues := strings.Split(configEnv, ",")
	for _, configValue := range configValues {
		svc := &Service{}
		if err = svc.loadConfig(configValue); err != nil {
			return
		}
		if err = svc.loadInfo(containerInfo, mon.hostname); err != nil {
			return
		}

		oldSvc, update := mon.services[svc.Hash()]
		if update {
			serviceEvents <- ServiceEvent{Heartbeat, svc}
		} else if update && *svc == *oldSvc {
			serviceEvents <- ServiceEvent{Update, svc}
		} else {
			serviceEvents <- ServiceEvent{Add, svc}
		}
		mon.services[svc.Hash()] = svc
	}
	return
}

func (mon *ServiceMonitor) removeContainer(serviceEvents chan ServiceEvent, containerId string) {
	remove := []string{}
	for hash, svc := range mon.services {
		if svc.ContainerId == containerId {
			remove = append(remove, hash)
		}
	}

	for _, hash := range remove {
		serviceEvents <- ServiceEvent{Remove, mon.services[hash]}
		delete(mon.services, hash)
	}
}

func (mon *ServiceMonitor) poll(serviceEvents chan ServiceEvent) (err error) {
	var containers []dockerclient.Container
	if containers, err = mon.client.ListContainers(false); err != nil {
		return err
	}

	containerIds := make(map[string]bool, len(containers))
	for _, container := range containers {
		mon.addContainer(serviceEvents, container.Id)
		containerIds[container.Id] = true
	}

	for id := range mon.containers {
		if _, ok := containerIds[id]; !ok {
			mon.removeContainer(serviceEvents, id)
		}
	}
	mon.containers = containerIds
	return nil
}

func (mon *ServiceMonitor) Listen(serviceEvents chan ServiceEvent) error {
	if !stateListening(&mon.state) {
		return errors.New("already listening")
	}

	mon.containers = make(map[string]bool)
	mon.services = make(map[string]*Service)
	containerEvents := make(chan ContainerEvent, 1)

	cb := func(e *dockerclient.Event, args ...interface{}) {
		if e.Status == "start" {
			containerEvents <- ContainerEvent{Add, e.Id}
		} else if e.Status == "die" {
			containerEvents <- ContainerEvent{Remove, e.Id}
		}
	}

	mon.poll(serviceEvents)
	mon.client.StartMonitorEvents(cb)

Loop:
	for {
		select {
		case e := <-containerEvents:
			switch e.State {
			case Add:
				mon.addContainer(serviceEvents, e.ContainerId)
			case Remove:
				mon.removeContainer(serviceEvents, e.ContainerId)
			}
		case <-time.After(mon.pollInterval):
			mon.poll(serviceEvents)
		case <-mon.stop:
			break Loop
		}
	}

	mon.client.StopAllMonitorEvents()
	for _, service := range mon.services {
		serviceEvents <- ServiceEvent{Remove, service}
	}
	close(serviceEvents)

	stateStopped(&mon.state)
	return nil
}

func (mon *ServiceMonitor) Stop() error {
	if !stateStopping(&mon.state) {
		return errors.New("not listening")
	}
	mon.stop <- true
	return nil
}
