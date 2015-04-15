package main

import (
	"fmt"
	"gopkg.in/BlueDragonX/go-log.v0"
	"gopkg.in/BlueDragonX/go-settings.v0"
	"os"
	"os/signal"
	"syscall"
)

var logger *log.Logger = log.NewOrExit()

func main() {
	options := ParseOptionsOrExit(os.Args)
	config := settings.LoadOrExit(options.Config)

	// set config values from cli options
	if options.Var != "" {
		config.Set("service.var", options.Var)
	}
	if len(options.Etcd) > 0 {
		config.Set("etcd.uris", options.Etcd)
	}
	if options.Docker != "" {
		config.Set("docker.uri", options.Docker)
	}
	if options.Prefix != "" {
		config.Set("etcd.prefix", options.Prefix)
	}
	if options.LogTarget != "" {
		config.Set("logging.target", options.LogTarget)
	}
	if options.LogLevel != "" {
		config.Set("logging.level", options.LogLevel)
	}

	// configure the logger
	if logTarget, err := config.String("logging.target"); err == nil {
		if logTargetObj, err := log.NewTarget(logTarget); err == nil {
			logger.SetTarget(logTargetObj)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	}
	if logLevel, err := config.String("logging.level"); err == nil {
		logger.SetLevel(log.NewLevel(logLevel))
	}

	// start the system
	var err error
	var mon *ServiceMonitor
	var ann *ServiceAnnouncer

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	if mon, err = ConfigServiceMonitor(config); err != nil {
		logger.Fatalf("monitor failed: %s", err)
	}
	if ann, err = ConfigServiceAnnouncer(config); err != nil {
		logger.Fatalf("announcer failed: %s", err)
	}

	events := make(chan *ServiceEvent, 1)
	finish := make(chan bool)

	go func() {
		if err = mon.Listen(events); err != nil {
			logger.Fatalf("failed to start monitor: %s", err)
		}
		finish <- true
	}()

	logger.Info("started")
Loop:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				break Loop
			}
			if err := ann.Announce(event); err == nil {
				logger.Debugf("event processed: %+v", event)
			} else {
				logger.Errorf("event error: %+v: %s", event, err)
			}
		case <-signals:
			if err = mon.Stop(); err != nil {
				logger.Errorf("%s", err)
			}
		}
	}

	<-finish
	logger.Info("stopped")
}
