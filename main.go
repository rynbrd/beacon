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
	if options.EnvVar != "" {
		config.Set("beacon.env-var", options.EnvVar)
	}
	if len(options.Etcd) > 0 {
		config.Set("etcd.uris", options.Etcd)
	}
	if options.Prefix != "" {
		config.Set("etcd.prefix", options.Prefix)
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
