package docker_test

import (
	docker "."
	"fmt"
	"github.com/BlueDragonX/beacon/beacon"
	"github.com/BlueDragonX/go-docker-test/dockertest"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"sync"
	"testing"
	"time"
)

const (
	DOCKER_IMAGE_REPO = "alpine"
	DOCKER_IMAGE_TAG  = "latest"
)

func WaitForEvents(ch <-chan *beacon.Event, n int, timeout time.Duration) ([]*beacon.Event, error) {
	events := make([]*beacon.Event, 0, n)
	timer := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case event, ok := <-ch:
			if !ok {
				return events, errors.New("channel closed")
			}
			events = append(events, event)
		case <-timer:
			return events, errors.New("timed out")
		}
	}
	return events, nil
}

func DockerSetup() (*dockertest.Docker, error) {
	daemon, err := dockertest.New()
	if err != nil {
		return nil, errors.Wrap(err, "unable to start docker daemon")
	}

	client, err := daemon.Client()
	if err != nil {
		daemon.Close()
		return nil, errors.Wrapf(err, "unable to pull docker image %s:%s", DOCKER_IMAGE_REPO, DOCKER_IMAGE_TAG)
	}
	err = client.PullImage(dockerclient.PullImageOptions{
		Repository: DOCKER_IMAGE_REPO,
		Tag:        DOCKER_IMAGE_TAG,
	}, dockerclient.AuthConfiguration{})
	if err != nil {
		daemon.Close()
		return nil, errors.Wrapf(err, "unable to pull docker image %s:%s", DOCKER_IMAGE_REPO, DOCKER_IMAGE_TAG)
	}
	return daemon, nil
}

// StartContainer starts a test container and returns its ID.
func StartContainer(daemon *dockertest.Docker, bindings []*beacon.Binding, labels map[string]string) (string, error) {
	client, err := daemon.Client()
	if err != nil {
		return "", errors.Wrap(err, "unable to create docker client")
	}

	exposedPorts := map[dockerclient.Port]struct{}{}
	for _, binding := range bindings {
		port := dockerclient.Port(fmt.Sprintf("%d/%s", binding.ContainerPort, binding.Protocol))
		exposedPorts[port] = struct{}{}
	}

	container, err := client.CreateContainer(dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image:        fmt.Sprintf("%s:%s", DOCKER_IMAGE_REPO, DOCKER_IMAGE_TAG),
			Cmd:          []string{"/bin/sh", "-c", "trap exit SIGTERM SIGINT; while true; do sleep 0.1; done"},
			ExposedPorts: exposedPorts,
			Labels:       labels,
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to create docker container")
	}

	portBindings := map[dockerclient.Port][]dockerclient.PortBinding{}
	for _, binding := range bindings {
		port := dockerclient.Port(fmt.Sprintf("%d/%s", binding.ContainerPort, binding.Protocol))
		portBindings[port] = []dockerclient.PortBinding{{
			HostIP:   binding.HostIP,
			HostPort: fmt.Sprintf("%d", binding.HostPort),
		}}
	}

	err = client.StartContainer(container.ID, &dockerclient.HostConfig{
		PortBindings: portBindings,
	})
	if err != nil {
		StopContainer(daemon, container.ID)
		return "", errors.Wrap(err, "unable to start docker container")
	}
	return container.ID, nil
}

// StopContainer stops a running container.
func StopContainer(daemon *dockertest.Docker, id string) error {
	client, err := daemon.Client()
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}

	err = client.RemoveContainer(dockerclient.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to stop container %s", id)
	}
	return nil
}

func ContainersEqual(a *beacon.Container, b *beacon.Container, ignoreID, ignoreHostPort bool) error {
	if !ignoreID && a.ID != b.ID {
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
		if (!ignoreHostPort && b1.HostPort != b2.HostPort) || b1.ContainerPort != b2.ContainerPort || b1.Protocol != b2.Protocol {
			return errors.Errorf("container.Bindings[%d] inequal: %+v != %+v", n, b1, b2)
		}
	}
	return nil
}

func EventsEqual(a *beacon.Event, b *beacon.Event, ignoreID, ignoreHostPort bool) error {
	if a.Action != b.Action {
		return errors.Errorf("event.Action inequal: %s != %s", a.Action, b.Action)
	}
	if err := ContainersEqual(a.Container, b.Container, ignoreID, ignoreHostPort); err != nil {
		return errors.Wrap(err, "event.Container inequal")
	}
	return nil
}

func EventArraysEqual(a []*beacon.Event, b []*beacon.Event, ignoreID, ignoreHostPort bool) error {
	if len(a) != len(b) {
		return errors.Errorf("event arrays have inequal length: %d != %d", len(a), len(b))
	}
	for n := range a {
		if err := EventsEqual(a[n], b[n], ignoreID, ignoreHostPort); err != nil {
			return errors.Wrapf(err, "events[%d] inequal", n)
		}
	}
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()
	daemon, err := dockertest.New()
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()

	hostIP := "10.1.1.100"
	runtime, err := docker.New(daemon.URL(), hostIP, "service", false)
	if err != nil {
		t.Error(err)
	} else if err := runtime.Close(); err != nil {
		t.Error(err)
	}
}

func TestOneContainer(t *testing.T) {
	t.Parallel()
	daemon, err := DockerSetup()
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()

	hostIP := "10.1.1.100"
	runtime, err := docker.New(daemon.URL(), hostIP, "service", false)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	ch, err := runtime.EmitEvents()
	if err != nil {
		t.Fatal(err)
	}

	wantService := "test"
	wantContainers := []*beacon.Container{
		{
			Service: wantService,
			Labels: map[string]string{
				"service": wantService,
				"test":    "TestOneContainer",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wantEvents := []*beacon.Event{}
	go func() {
		defer wg.Done()
		for _, container := range wantContainers {
			if id, err := StartContainer(daemon, container.Bindings, container.Labels); err != nil {
				t.Fatal(err)
			} else {
				container.ID = id
				t.Logf("container %s started", container.ID)
				wantEvents = append(wantEvents, &beacon.Event{
					Action:    beacon.Start,
					Container: container,
				})
			}
		}
		time.Sleep(5 * time.Second)
		for _, container := range wantContainers {
			if err := StopContainer(daemon, container.ID); err != nil {
				t.Fatal(err)
			}
			t.Logf("container %s stopped", container.ID)
			wantEvents = append(wantEvents, &beacon.Event{
				Action:    beacon.Stop,
				Container: &beacon.Container{ID: container.ID},
			})
		}
	}()

	haveEvents, err := WaitForEvents(ch, len(wantContainers)*2, 30*time.Second)
	if err != nil {
		t.Fatalf("failed waiting for events, received %d: %s", len(haveEvents), err)
	}
	wg.Wait()

	if err := EventArraysEqual(haveEvents, wantEvents, false, true); err != nil {
		t.Error(err)
	}
	for _, event := range haveEvents {
		for _, binding := range event.Container.Bindings {
			if binding.HostPort == 0 {
				t.Errorf("event {Action:%s Container:%s} has binding with HostPort of 0", event.Action, event.Container.ID)
			}
		}
	}
}

func TestTwoContainers(t *testing.T) {
	t.Parallel()
	daemon, err := DockerSetup()
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()

	hostIP := "10.1.1.100"
	runtime, err := docker.New(daemon.URL(), hostIP, "service", false)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	ch, err := runtime.EmitEvents()
	if err != nil {
		t.Fatal(err)
	}

	wantService := "test"
	wantContainers := []*beacon.Container{
		{
			Service: wantService,
			Labels: map[string]string{
				"service": wantService,
				"test":    "TestTwoContainers",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
		{
			Service: wantService,
			Labels: map[string]string{
				"service": wantService,
				"test":    "TestTwoContainers",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wantEvents := []*beacon.Event{}
	go func() {
		defer wg.Done()
		for _, container := range wantContainers {
			if id, err := StartContainer(daemon, container.Bindings, container.Labels); err != nil {
				t.Fatal(err)
			} else {
				container.ID = id
				t.Logf("container %s started", container.ID)
				wantEvents = append(wantEvents, &beacon.Event{
					Action:    beacon.Start,
					Container: container,
				})
			}
		}
		time.Sleep(5 * time.Second)
		for _, container := range wantContainers {
			if err := StopContainer(daemon, container.ID); err != nil {
				t.Fatal(err)
			}
			t.Logf("container %s stopped", container.ID)
			wantEvents = append(wantEvents, &beacon.Event{
				Action:    beacon.Stop,
				Container: &beacon.Container{ID: container.ID},
			})
		}
	}()

	haveEvents, err := WaitForEvents(ch, len(wantContainers)*2, 30*time.Second)
	if err != nil {
		t.Fatalf("failed waiting for events, received %d: %s", len(haveEvents), err)
	}
	wg.Wait()

	if err := EventArraysEqual(haveEvents, wantEvents, false, true); err != nil {
		t.Error(err)
	}
	for _, event := range haveEvents {
		for _, binding := range event.Container.Bindings {
			if binding.HostPort == 0 {
				t.Errorf("event {Action:%s Container:%s} has binding with HostPort of 0", event.Action, event.Container.ID)
			}
		}
	}
}

func TestRunIgnoredContainers(t *testing.T) {
	t.Parallel()
	daemon, err := DockerSetup()
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()

	hostIP := "10.1.1.100"
	runtime, err := docker.New(daemon.URL(), hostIP, "service", false)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	ch, err := runtime.EmitEvents()
	if err != nil {
		t.Fatal(err)
	}

	wantService := "test"
	wantContainers := []*beacon.Container{
		{
			Service: wantService,
			Labels: map[string]string{
				"service": wantService,
				"test":    "TestTwoContainers",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
		{
			Service: wantService,
			Labels: map[string]string{
				"test": "TestTwoContainers",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wantEvents := []*beacon.Event{}
	go func() {
		defer wg.Done()
		for _, container := range wantContainers {
			if id, err := StartContainer(daemon, container.Bindings, container.Labels); err != nil {
				t.Fatal(err)
			} else {
				container.ID = id
				t.Logf("container %s started", container.ID)
				if _, ok := container.Labels["service"]; ok {
					wantEvents = append(wantEvents, &beacon.Event{
						Action:    beacon.Start,
						Container: container,
					})
				}
			}
		}
		time.Sleep(5 * time.Second)
		for _, container := range wantContainers {
			if err := StopContainer(daemon, container.ID); err != nil {
				t.Fatal(err)
			}
			t.Logf("container %s stopped", container.ID)
			if _, ok := container.Labels["service"]; ok {
				wantEvents = append(wantEvents, &beacon.Event{
					Action:    beacon.Stop,
					Container: &beacon.Container{ID: container.ID},
				})
			}
		}
	}()

	haveEvents, err := WaitForEvents(ch, 2, 30*time.Second)
	if err != nil {
		t.Fatalf("failed waiting for events, received %d: %s", len(haveEvents), err)
	}
	wg.Wait()

	if err := EventArraysEqual(haveEvents, wantEvents, false, true); err != nil {
		t.Error(err)
	}
	for _, event := range haveEvents {
		for _, binding := range event.Container.Bindings {
			if binding.HostPort == 0 {
				t.Errorf("event {Action:%s Container:%s} has binding with HostPort of 0", event.Action, event.Container.ID)
			}
		}
	}
}

func TestExistingContainer(t *testing.T) {
	t.Parallel()
	daemon, err := DockerSetup()
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()

	hostIP := "10.1.1.100"
	runtime, err := docker.New(daemon.URL(), hostIP, "service", false)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	wantService := "test"
	wantContainers := []*beacon.Container{
		{
			Service: wantService,
			Labels: map[string]string{
				"service": wantService,
				"test":    "TestExistingContainer",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wantEvents := []*beacon.Event{}
	go func() {
		defer wg.Done()
		for _, container := range wantContainers {
			if id, err := StartContainer(daemon, container.Bindings, container.Labels); err != nil {
				t.Fatal(err)
			} else {
				container.ID = id
				t.Logf("container %s started", container.ID)
				wantEvents = append(wantEvents, &beacon.Event{
					Action:    beacon.Start,
					Container: container,
				})
			}
		}
	}()
	wg.Wait()

	ch, err := runtime.EmitEvents()
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Second)
		for _, container := range wantContainers {
			if err := StopContainer(daemon, container.ID); err != nil {
				t.Fatal(err)
			}
			t.Logf("container %s stopped", container.ID)
			wantEvents = append(wantEvents, &beacon.Event{
				Action:    beacon.Stop,
				Container: &beacon.Container{ID: container.ID},
			})
		}
	}()

	haveEvents, err := WaitForEvents(ch, len(wantContainers)*2, 30*time.Second)
	if err != nil {
		t.Fatalf("failed waiting for events, received %d: %s", len(haveEvents), err)
	}
	wg.Wait()

	if err := EventArraysEqual(haveEvents, wantEvents, false, true); err != nil {
		t.Error(err)
	}
	for _, event := range haveEvents {
		for _, binding := range event.Container.Bindings {
			if binding.HostPort == 0 {
				t.Errorf("event {Action:%s Container:%s} has binding with HostPort of 0", event.Action, event.Container.ID)
			}
		}
	}
}

func TestStopOnClose(t *testing.T) {
	t.Parallel()
	daemon, err := DockerSetup()
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()

	hostIP := "10.1.1.100"
	runtime, err := docker.New(daemon.URL(), hostIP, "service", true)
	if err != nil {
		t.Fatal(err)
	}

	ch, err := runtime.EmitEvents()
	if err != nil {
		t.Fatal(err)
	}

	wantService := "test"
	wantContainers := []*beacon.Container{
		{
			Service: wantService,
			Labels: map[string]string{
				"service": wantService,
				"test":    "TestOneContainer",
			},
			Bindings: []*beacon.Binding{
				{HostIP: "127.0.0.1", HostPort: 0, ContainerPort: 80, Protocol: beacon.TCP},
			},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wantEvents := []*beacon.Event{}
	go func() {
		defer wg.Done()
		for _, container := range wantContainers {
			if id, err := StartContainer(daemon, container.Bindings, container.Labels); err != nil {
				t.Fatal(err)
			} else {
				container.ID = id
				t.Logf("container %s started", container.ID)
				wantEvents = append(wantEvents, &beacon.Event{
					Action:    beacon.Start,
					Container: container,
				}, &beacon.Event{
					Action: beacon.Stop,
					Container: &beacon.Container{
						ID: id,
					},
				})
			}
		}
		time.Sleep(5 * time.Second)
		runtime.Close()
	}()

	haveEvents, err := WaitForEvents(ch, len(wantContainers)*2, 30*time.Second)
	if err != nil {
		t.Fatalf("failed waiting for events, received %d: %s", len(haveEvents), err)
	}
	wg.Wait()

	if err := EventArraysEqual(haveEvents, wantEvents, false, true); err != nil {
		t.Error(err)
	}
	for _, event := range haveEvents {
		for _, binding := range event.Container.Bindings {
			if binding.HostPort == 0 {
				t.Errorf("event {Action:%s Container:%s} has binding with HostPort of 0", event.Action, event.Container.ID)
			}
		}
	}
}
