package main

import (
	"github.com/BlueDragonX/beacon/beacon"
	"github.com/BlueDragonX/beacon/debug"
	"github.com/BlueDragonX/beacon/docker"
	"github.com/BlueDragonX/beacon/sns"
	"github.com/pkg/errors"
	"log"
	"os"
)

// Logger is the application logger.
var Logger = log.New(os.Stdout, "", 0)

func init() {
	beacon.Logger = Logger
	docker.Logger = Logger
}

// NewBeacon creates a new Beacon from configuration.
func NewBeacon(config *Config) (beacon.Beacon, error) {
	docker, err := docker.New(
		config.Docker.Socket,
		config.Docker.HostIP,
		config.Docker.Label,
		config.Docker.StopOnExit,
	)
	if err != nil {
		return nil, err
	}

	routes := make([]beacon.Route, len(config.Backends))
	for n, backendCfg := range config.Backends {
		var backend beacon.Backend
		filter := beacon.NewFilter(backendCfg.Filter)
		if backendCfg.SNS != nil {
			backend = sns.New(
				backendCfg.SNS.Region,
				backendCfg.SNS.Topic,
			)
		} else if backendCfg.Debug != nil {
			backend = debug.New(Logger)
		} else {
			return nil, errors.New("unsupported backend")
		}
		routes[n] = beacon.NewRoute(filter, backend)
	}
	return beacon.New(docker, routes)
}

func main() {
	config := Configure(os.Args)
	bcn, err := NewBeacon(config)
	if err != nil {
		Logger.Fatalf("failed to initialize: %s", err)
	}

	signals := notifyOnStop()

	Logger.Printf("listening for events on %s", config.Docker.Socket)
	go func() {
		<-signals
		if err := bcn.Close(); err != nil {
			Logger.Fatalf("failed to shut down: %s", err)
		}
	}()

	if err := bcn.Run(); err != nil {
		Logger.Fatalf("failed to shut down: %s", err)
	}
	Logger.Print("stopped")
}
