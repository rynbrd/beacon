// Beacon should announce a service address to its Discovery backend when a
// Listener emits an Add. Multiple events should result in multiple adds. A
// subsequent add with the same information should not trigger an announcement.
// All adds should have a TTL argument equal to `beacon.Heartbeat + beacon.TTL`.
//
// Beacon should shutdown a service address in its Discovery backend when a
// Listener emits a Remove. Multiple events should results in multiple removes.
// The removal of a missing address should succeed.
//
// Beacon should re-announce all services at an interval of `beacon.Heartbeat`.
//
// Beacon should shutdown all services on close.
package main

import (
	"testing"
	"time"
	"strings"
)

func mustParseAddress(t *testing.T, address string) *Address {
	addr, err := ParseAddress(address)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func mustParseMapping(t *testing.T, mapping string) *Mapping {
	parts := strings.SplitN(mapping, "->", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid mapping %s", mapping)
	}
	addr, err := ParseAddress(parts[0])
	if err != nil {
		t.Fatal("invalid mapping address %s", parts[0])
	}
	port, err := ParsePort(parts[1])
	if err != nil {
		t.Fatal("invalid mapping port %s", parts[1])
	}
	return &Mapping{addr, port}
}

func mustParseMappings(t *testing.T, mappingsStr string) []*Mapping {
	mappings := []*Mapping{}
	for _, part := range strings.Split(mappingsStr, ",") {
		mappings = append(mappings, mustParseMapping(t, part))
	}
	return mappings
}

type BeaconInput struct {
	action    ContainerAction
	name      string
	container *Container
	addr      *Address
}

func testBeacon(t *testing.T, inputs []BeaconInput, announcements, shutdowns int) {
	listening := make(chan bool)
	listener := NewMockListener(listening)
	actions := make(chan ServiceAction, announcements + shutdowns + len(inputs) + 1)
	discovery := NewMockDiscovery(actions)
	beacon := &Beacon{
		Heartbeat: 30 * time.Second,
		TTL: 30 * time.Second,
		EnvVar: "SERVICES",
		Listeners: []Listener{listener},
		Discovery: discovery,
	}

	defer close(listening)
	defer close(actions)
	ttl := 60 * time.Second

	go func() {
		// wait for the listener to come online
		select {
		case isListening := <-listening:
			if !isListening {
				t.Fatal("got false from listening, this shouldn't happen")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timed out waiting for listener")
		}

		// add/remove containers
		services := make(map[MockDiscoveryKey]MockDiscoveryValue, len(inputs))
		for _, input := range inputs {
			if input.action == ContainerAdd {
				key := MockDiscoveryKey{input.name, input.container.ID}
				value := MockDiscoveryValue{input.addr, ttl, 1}
				services[key] = value

				t.Logf("emiting add for %+v\n", input.container)
			} else if input.action == ContainerRemove {
				key := MockDiscoveryKey{input.name, input.container.ID}
				delete(services, key)
				t.Logf("emiting remove for %+v\n", input.container)
			}
			listener.Emit(&ContainerEvent{input.action, input.container})
		}

		// verify services
		announceCalls := 0
		shutdownCalls := 0
		for i := 0; i < announcements + shutdowns; i++ {
			select {
			case action := <-actions:
				if action == ServiceAnnounce {
					announceCalls += 1
				} else if action == ServiceShutdown {
					shutdownCalls += 1
				}
			case <-time.After(1 * time.Second):
				t.Errorf("announce/shutdown not called %d times", announcements)
				break
			}
		}
		if announceCalls != announcements {
			t.Error("announce called %d times, not %d", announceCalls, announcements)
		}
		if shutdownCalls != shutdowns {
			t.Error("shutdown called %d times, not %d", shutdownCalls, shutdowns)
		}
		if err := MockServicesEqual(services, discovery.Services, false); err != nil {
			t.Error(err)
			t.Errorf("  want: %+v", services)
			t.Errorf("  have: %+v", discovery.Services)
		}

		// close beacon and wait for the listener
		beacon.Close()
		select {
		case isListening := <-listening:
			if isListening {
				t.Fatal("got true from listening, this shouldn't happen")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timed out waiting for listener")
		}
	}()

	beacon.Run()
}

func TestBeaconAddOne(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
	}
	testBeacon(t, inputs, 1, 0)
}

func TestBeaconAddDuplicate(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
	}
	testBeacon(t, inputs, 1, 0)
}

func TestBeaconAddMultiple(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerAdd, "radius",
		 &Container{"c2", []string{"SERVICES=radius:1643/udp"}, "172.16.0.11", mustParseMappings(t, "10.1.1.100:49001/udp->1643/udp")},
		 mustParseAddress(t, "10.1.1.100:49001/udp")},
		{ContainerAdd, "api",
		 &Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
		 mustParseAddress(t, "172.16.0.12:443/tcp")},
	}
	testBeacon(t, inputs, 3, 0)
}

func TestBeaconRemoveOne(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerRemove, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
	}
	testBeacon(t, inputs, 1, 1)
}

func TestBeaconRemoveDuplicate(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerRemove, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerRemove, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
	}
	testBeacon(t, inputs, 1, 1)
}

func TestBeaconRemoveMultiple(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerAdd, "radius",
		 &Container{"c2", []string{"SERVICES=radius:1643/udp"}, "172.16.0.11", mustParseMappings(t, "10.1.1.100:49001/udp->1643/udp")},
		 mustParseAddress(t, "10.1.1.100:49001/udp")},
		{ContainerAdd, "api",
		 &Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
		 mustParseAddress(t, "172.16.0.12:443/tcp")},
		{ContainerRemove, "www",
		 &Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		 mustParseAddress(t, "10.1.1.100:49000/tcp")},
		{ContainerRemove, "api",
		 &Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
		 mustParseAddress(t, "172.16.0.12:443/tcp")},
	}
	testBeacon(t, inputs, 3, 2)
}

func TestBeaconHeartbeatAndClose(t *testing.T) {
	listening := make(chan bool)
	listener := NewMockListener(listening)
	discovery := NewMockDiscovery(nil)
	beacon := &Beacon{
		Heartbeat: 2 * time.Second,
		TTL: 30 * time.Second,
		EnvVar: "SERVICES",
		Listeners: []Listener{listener},
		Discovery: discovery,
	}

	defer close(listening)

	containers := []*Container{
		&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		&Container{"c2", []string{"SERVICES=radius:1643/udp"}, "172.16.0.11", mustParseMappings(t, "10.1.1.100:49001/udp->1643/udp")},
		&Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
	}

	go func() {
		// wait for the listener to come online
		select {
		case isListening := <-listening:
			if !isListening {
				t.Fatal("got false from listening, this shouldn't happen")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timed out waiting for listener")
		}

		for _, container := range containers {
			listener.Emit(&ContainerEvent{ContainerAdd, container})
		}
		time.Sleep(3 * time.Second)

		if len(discovery.Services) != len(containers) {
			t.Error("wrong number of services announced")
		}
		for key, value := range discovery.Services {
			if value.Count != 2 {
				t.Errorf("no heartbeat for %+v:%+v", key, value)
			}
		}

		// close beacon and wait for the listener
		beacon.Close()
		select {
		case isListening := <-listening:
			if isListening {
				t.Fatal("got true from listening, this shouldn't happen")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timed out waiting for listener")
		}
	}()

	beacon.Run()

	if len(discovery.Services) != 0 {
		t.Errorf("services not shutdown on close: %+v", discovery.Services)
	}
}
