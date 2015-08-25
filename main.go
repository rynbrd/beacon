package main

import (
	"fmt"
	"gopkg.in/BlueDragonX/go-log.v1"
	"gopkg.in/BlueDragonX/go-settings.v1"
	"os"
	"os/signal"
	"syscall"
)

var logger *log.Logger = log.NewOrExit()

func configure(args []string) *settings.Settings {
	// load configuration
	options := ParseOptionsOrExit(args)
	var config *settings.Settings
	if _, err := os.Stat(options.Config); os.IsNotExist(err) {
		config = settings.New()
	} else if err == nil {
		config = settings.LoadOrExit(options.Config)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// set config values from cli options
	if options.Hostname != "" {
		config.Set("beacon.hostname", options.Hostname)
	}
	if options.EnvVar != "" {
		config.Set("beacon.env-var", options.EnvVar)
	}
	if options.Heartbeat.Nanoseconds() != 0 {
		config.Set("beacon.heartbeat", options.Heartbeat)
	}
	if options.TTL.Nanoseconds() != 0 {
		config.Set("beacon.ttl", options.TTL)
	}
	if len(options.Etcd) > 0 {
		config.Set("etcd.uris", options.Etcd)
	}
	if options.EtcdPrefix != "" {
		config.Set("etcd.prefix", options.EtcdPrefix)
	}
	if options.EtcdFormat != "" {
		config.Set("etcd.format", options.EtcdFormat)
	}
	if options.Docker != "" {
		config.Set("docker.uri", options.Docker)
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
	return config
}

func main() {
	config := configure(os.Args)
	beacon := ConfigBeacon(config)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-signals
		if err := beacon.Close(); err != nil {
			logger.Fatal(err.Error())
		}
	}()
	beacon.Run()
}
