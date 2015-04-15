package main

import (
	"fmt"
	"gopkg.in/BlueDragonX/go-settings.v0"
	"gopkg.in/BlueDragonX/simplelog.v1"
	"os"
	"os/signal"
	"syscall"
)

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

	var log *simplelog.Logger
	var mon *ServiceMonitor
	var ann *ServiceAnnouncer
	var err error

	if log, err = simplelog.NewLogger(simplelog.CONSOLE, "beacon"); err != nil {
		fmt.Println("failed to create logger:", err)
		os.Exit(1)
	}

	log.SetLevel(simplelog.StringToLevel(config.StringDflt("logging.level", "info")))
	log.Notice("starting")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	if mon, err = ConfigServiceMonitor(config, log); err != nil {
		log.Fatal("monitor failed: %s", err)
	}
	if ann, err = ConfigServiceAnnouncer(config, log); err != nil {
		log.Fatal("announcer failed: %s", err)
	}

	events := make(chan *ServiceEvent, 1)
	finish := make(chan bool)

	go func() {
		if err = mon.Listen(events); err != nil {
			log.Fatal("failed to start monitor: %s", err)
		}
		finish <- true
	}()

	log.Notice("started")
Loop:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				break Loop
			}
			if err := ann.Announce(event); err == nil {
				log.Info("event processed: %s", event)
			} else {
				log.Error("event error: %s: %s", event, err)
			}
		case <-signals:
			log.Notice("stopping")
			if err = mon.Stop(); err != nil {
				log.Warn("%s", err)
			}
		}
	}

	<-finish
	log.Notice("stopped")
}
