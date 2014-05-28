package main

import (
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	Listening = iota
	Stopping
	Stopped
)

// Parse an environment variable into a name and value.
func parseEnv(envVar string) (name string, value string) {
	parts := strings.SplitN(envVar, "=", 2)
	name = parts[0]
	if len(parts) == 2 {
		value = parts[1]
	}
	return
}

// Parse a port declaration. Protocol is assumed to be tcp if absent.
func parsePort(portStr string) (port int, protocol string, err error) {
	parts := strings.SplitN(portStr, "/", 2)
	if port, err = strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
		return
	}

	if len(parts) == 1 {
		protocol = "tcp"
	} else {
		protocol = strings.ToLower(strings.TrimSpace(parts[1]))
	}
	return
}

// Stop a routing that's using a chan chan bool to interrupt it.
func stopRoutine(control chan chan bool) {
	done := make(chan bool)
	control <- done
	<-done
}

// Transition state to listening.
func stateListening(state *int32) bool {
	return atomic.CompareAndSwapInt32(state, Stopped, Listening)
}

// Transition state to stopping.
func stateStopping(state *int32) bool {
	return atomic.CompareAndSwapInt32(state, Listening, Stopping)
}

// Transition state to stopped.
func stateStopped(state *int32) bool {
	return atomic.CompareAndSwapInt32(state, Stopping, Stopped)
}
