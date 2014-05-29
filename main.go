package main

import (
	"fmt"
	"gopkg.in/BlueDragonX/simplelog.v1"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var log *simplelog.Logger
	var mon *ServiceMonitor
	var ann *ServiceAnnouncer
	var cfg Config
	var err error

	if log, err = simplelog.NewLogger(simplelog.CONSOLE, "bosun"); err != nil {
		fmt.Println("failed to create logger:", err)
		os.Exit(1)
	}
	if cfg, err = LoadConfig(); err != nil {
		log.Fatal("error parsing config: %s", err)
	}
	if errs := cfg.Validate(); len(errs) != 0 {
		log.Error("config file is invalid:")
		for _, err = range errs {
			log.Error("  %s", err)
		}
		log.Fatal("could not process config file")
	}

	log.Notice("starting")
	log.Info("using Docker at %s", cfg.Docker.URL)
	log.Info("using etcd at %s", strings.Join(cfg.Etcd.URLs, ", "))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	if mon, err = NewServiceMonitor(cfg.Docker.URL, cfg.Docker.Hostname, cfg.Docker.Var, cfg.Etcd.Heartbeat); err != nil {
		log.Fatal("monitor failed: %s", err)
	}

	ttl := uint64(cfg.Etcd.Heartbeat.Seconds() + cfg.Etcd.TTL.Seconds())
	if cfg.Etcd.IsTLS() {
		ann, err = NewTLSServiceAnnouncer(cfg.Etcd.URLs, cfg.Etcd.TLSCert, cfg.Etcd.TLSKey, cfg.Etcd.TLSCACert, cfg.Etcd.Prefix, ttl)
		if err != nil {
			log.Fatal("announcer failed: %s", err)
		}
	} else {
		ann = NewServiceAnnouncer(cfg.Etcd.URLs, cfg.Etcd.Prefix, ttl)
	}

	events := make(chan ServiceEvent, 1)
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
				log.Info("event error: %s: %s", event, err)
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
