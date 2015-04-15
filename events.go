package main

type ContainerAction int
type ServiceAction int

const (
	// Container event actions.
	ContainerAdd ContainerAction = iota
	ContainerRemove
)

const (
	// Service event actions.
	ServiceAdd ServiceAction = iota
	ServiceRemove
	ServiceUpdate
	ServiceHeartbeat
)

type ContainerEvent struct {
	Action      ContainerAction
	ContainerId string
}

type ServiceEvent struct {
	Action  ServiceAction
	Service *Service
}
