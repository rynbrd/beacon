package main

import (
	"fmt"
	"time"
)

type ServiceAction int

const (
	ServiceAnnounce ServiceAction = iota
	ServiceShutdown
)

// A mock which implements the Discovery interface.
type MockDiscovery struct {
	Services map[MockDiscoveryKey]MockDiscoveryValue
	actions  chan ServiceAction
}

func NewMockDiscovery(actions chan ServiceAction) *MockDiscovery {
	return &MockDiscovery{map[MockDiscoveryKey]MockDiscoveryValue{}, actions}
}

// Announce creates and adds a key/value pair to the Services map. No error is returned.
func (m *MockDiscovery) Announce(name, container string, address *Address, ttl time.Duration) error {
	key := MockDiscoveryKey{name, container}
	value, has := m.Services[key]
	if has && value.Address.Equal(address) && value.TTL == ttl {
		value.Count += 1
	} else {
		value = MockDiscoveryValue{address, ttl, 1}
	}
	m.Services[key] = value
	if m.actions != nil {
		m.actions <- ServiceAnnounce
	}
	return nil
}

// Shutdown removes a key from the Service map. No error is returned.
func (m *MockDiscovery) Shutdown(name, container string) error {
	delete(m.Services, MockDiscoveryKey{name, container})
	if m.actions != nil {
		m.actions <- ServiceShutdown
	}
	return nil
}

// Close does nothing and returns nil.
func (m *MockDiscovery) Close() error {
	return nil
}

// MockDiscoveryKey is used as the key for the Services map in MockDiscovery.
type MockDiscoveryKey struct {
	Name      string
	Container string
}

// MockDiscoveryValue is used as the value for the Services map in MockDiscovery.
type MockDiscoveryValue struct {
	Address *Address
	TTL     time.Duration
	Count   int
}

// Return an error if two mock service maps are not equal.
func MockServicesEqual(left, right map[MockDiscoveryKey]MockDiscoveryValue, ignoreCount bool) error {
	leftLen := len(left)
	rightLen := len(right)
	if leftLen != rightLen {
		return fmt.Errorf("lengths differ: %d != %d", leftLen, rightLen)
	}

	for key, leftVal := range left {
		rightVal, has := right[key]
		if !has {
			return fmt.Errorf("key %+v does not exist in right", key)
		}
		if leftVal.Address == nil && rightVal.Address != nil || !leftVal.Address.Equal(rightVal.Address) {
			return fmt.Errorf("address of %+v not equal: %+v != %+v", key, leftVal.Address, rightVal.Address)
		}
		if leftVal.TTL != rightVal.TTL {
			return fmt.Errorf("ttl of %+v not equal: %s != %s", key, leftVal.TTL, rightVal.TTL)
		}
		if !ignoreCount && leftVal.Count != rightVal.Count {
			return fmt.Errorf("count of %+v not equal: %d != %d", key, leftVal.Count, rightVal.Count)
		}
	}
	return nil
}
