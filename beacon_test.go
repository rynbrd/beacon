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
	"strings"
	"testing"
	"time"
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
	container *Container
	services  []BeaconService
}

type BeaconService struct {
	name string
	addr *Address
}

func testBeacon(t *testing.T, inputs []BeaconInput, announcements, shutdowns int) {
	actionLen := announcements + shutdowns + 1
	for _, input := range inputs {
		actionLen += len(input.services)
	}
	actions := make(chan ServiceAction, actionLen)
	defer close(actions)
	discovery := NewMockDiscovery(actions)

	listening := make(chan bool)
	listener := NewMockListener(listening)
	defer close(listening)

	beacon := &Beacon{
		Heartbeat: 30 * time.Second,
		TTL:       30 * time.Second,
		EnvVar:    "SERVICES",
		Listeners: []Listener{listener},
		Discovery: discovery,
	}

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
			for _, inSvc := range input.services {
				if input.action == ContainerAdd {
					key := MockDiscoveryKey{inSvc.name, input.container.ID}
					value := MockDiscoveryValue{inSvc.addr, ttl, 1}
					services[key] = value

					t.Logf("emiting add for %+v\n", input.container)
				} else if input.action == ContainerRemove {
					key := MockDiscoveryKey{inSvc.name, input.container.ID}
					delete(services, key)
					t.Logf("emiting remove for %+v\n", input.container)
				}
			}
			listener.Emit(&ContainerEvent{input.action, input.container})
		}

		// verify services
		announceCalls := 0
		shutdownCalls := 0
		for i := 0; i < announcements+shutdowns; i++ {
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
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
	}
	testBeacon(t, inputs, 1, 0)
}

func TestBeaconAddDuplicate(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
	}
	testBeacon(t, inputs, 1, 0)
}

func TestBeaconAddMultipleContainers(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerAdd,
			&Container{"c2", []string{"SERVICES=radius:1643/udp"}, "172.16.0.11", mustParseMappings(t, "10.1.1.100:49001/udp->1643/udp")},
			[]BeaconService{{"radius", mustParseAddress(t, "10.1.1.100:49001/udp")}}},
		{ContainerAdd,
			&Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
			[]BeaconService{{"api", mustParseAddress(t, "172.16.0.12:443/tcp")}}},
	}
	testBeacon(t, inputs, 3, 0)
}

func TestBeaconNoService(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{}},
	}
	testBeacon(t, inputs, 0, 0)
}

func TestBeaconAddMultipleServices(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80,www-ssl:443"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp,10.1.1.100:49001/tcp->443/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}, {"www-ssl", mustParseAddress(t, "10.1.1.100:49001/tcp")}}},
		{ContainerAdd,
			&Container{"c2", []string{"SERVICES=www:80,www-ssl:443"}, "172.16.0.11", mustParseMappings(t, "10.1.1.101:49000/tcp->443/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "172.16.0.11:80/tcp")}, {"www-ssl", mustParseAddress(t, "10.1.1.101:49000/tcp")}}},
	}
	testBeacon(t, inputs, 4, 0)
}

func TestRemoveMultipleServices(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80,www-ssl:443"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp,10.1.1.100:49001/tcp->443/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}, {"www-ssl", mustParseAddress(t, "10.1.1.100:49001/tcp")}}},
		{ContainerRemove,
			&Container{"c1", []string{"SERVICES=www:80,www-ssl:443"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp,10.1.1.100:49001/tcp->443/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}, {"www-ssl", mustParseAddress(t, "10.1.1.100:49001/tcp")}}},
	}
	testBeacon(t, inputs, 2, 2)
}

func TestBeaconRemoveOne(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerRemove,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
	}
	testBeacon(t, inputs, 1, 1)
}

func TestBeaconRemoveDuplicate(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerRemove,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerRemove,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
	}
	testBeacon(t, inputs, 1, 1)
}

func TestBeaconRemoveMultipleContainers(t *testing.T) {
	inputs := []BeaconInput{
		{ContainerAdd,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerAdd,
			&Container{"c2", []string{"SERVICES=radius:1643/udp"}, "172.16.0.11", mustParseMappings(t, "10.1.1.100:49001/udp->1643/udp")},
			[]BeaconService{{"radius", mustParseAddress(t, "10.1.1.100:49001/udp")}}},
		{ContainerAdd,
			&Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
			[]BeaconService{{"api", mustParseAddress(t, "172.16.0.12:443/tcp")}}},
		{ContainerRemove,
			&Container{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
			[]BeaconService{{"www", mustParseAddress(t, "10.1.1.100:49000/tcp")}}},
		{ContainerRemove,
			&Container{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
			[]BeaconService{{"api", mustParseAddress(t, "172.16.0.12:443/tcp")}}},
	}
	testBeacon(t, inputs, 3, 2)
}

func TestBeaconHeartbeatAndClose(t *testing.T) {
	listening := make(chan bool)
	listener := NewMockListener(listening)
	discovery := NewMockDiscovery(nil)
	beacon := &Beacon{
		Heartbeat: 2 * time.Second,
		TTL:       30 * time.Second,
		EnvVar:    "SERVICES",
		Listeners: []Listener{listener},
		Discovery: discovery,
	}

	defer close(listening)

	containers := []*Container{
		{"c1", []string{"SERVICES=www:80"}, "172.16.0.10", mustParseMappings(t, "10.1.1.100:49000/tcp->80/tcp")},
		{"c2", []string{"SERVICES=radius:1643/udp"}, "172.16.0.11", mustParseMappings(t, "10.1.1.100:49001/udp->1643/udp")},
		{"c3", []string{"SERVICES=api:443/tcp"}, "172.16.0.12", []*Mapping{}},
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
