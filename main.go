package main

import (
	"fmt"
	"github.com/BlueDragonX/simplelog"
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
	if err = cfg.Validate(); err != nil {
		log.Fatal("invalid config: %s", err)
	}

	log.Notice("starting")
	log.Info("using Docker at %s", cfg.DockerURL)
	log.Info("using etcd at %s", strings.Join(cfg.EtcdURLs, ", "))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	if mon, err = NewServiceMonitor(cfg.DockerURL, cfg.Hostname, cfg.DockerVar, cfg.Heartbeat); err != nil {
		log.Fatal("monitor failed: %s", err)
	}

	if cfg.IsTLS() {
		ann, err = NewTLSServiceAnnouncer(cfg.EtcdURLs, cfg.TLSCert, cfg.TLSKey, cfg.TLSCACert, cfg.EtcdPrefix, uint64(cfg.TTL.Seconds()))
		if err != nil {
			log.Fatal("announcer failed: %s", err)
		}
	} else {
		ann = NewServiceAnnouncer(cfg.EtcdURLs, cfg.EtcdPrefix, uint64(cfg.TTL.Seconds()))
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
