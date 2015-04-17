package main

import (
	"flag"
	"strings"
	"time"
)

var DefaultConfigFile string = "/etc/beacon.yml"

// A string array option capable of being appended to.
type stringsOpt []string

func (strs *stringsOpt) String() string {
	return strings.Join(*strs, ",")
}

func (strs *stringsOpt) Set(value string) error {
	*strs = append(*strs, strings.Split(value, ",")...)
	return nil
}

// Store values retrieved from the cli.
type Options struct {
	Config    string
	EnvVar    string
	Heartbeat time.Duration
	TTL       time.Duration
	Etcd      []string
	Prefix    string
	Docker    string
	LogTarget string
	LogLevel  string
}

func ParseOptionsOrExit(args []string) *Options {
	var config, envVar, docker, prefix, logTarget, logLevel string
	var heartbeat, ttl time.Duration
	var etcd stringsOpt

	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.StringVar(&config, "config", DefaultConfigFile, "The path to the config file.")
	flags.StringVar(&envVar, "var", "", "The name of the service variable.")
	flags.DurationVar(&heartbeat, "heartbeat", 0*time.Second, "How often to refresh service TTL's.")
	flags.DurationVar(&ttl, "ttl", 0*time.Second, "How long to keep a service after missing a heartbeat.")
	flags.Var(&etcd, "etcd", "The etcd endpoint. May be provided multiple times.")
	flags.StringVar(&prefix, "prefix", "", "A prefix to prepend to all etcd key paths.")
	flags.StringVar(&docker, "docker", "", "The Docker endpoint.")
	flags.StringVar(&logTarget, "log-target", "", "The target to log to.")
	flags.StringVar(&logLevel, "log-level", "", "The level of logs to log.")
	flags.Parse(args[1:])

	return &Options{
		Config:    config,
		EnvVar:    envVar,
		Heartbeat: heartbeat,
		TTL:       ttl,
		Etcd:      []string(etcd),
		Prefix:    prefix,
		Docker:    docker,
		LogTarget: logTarget,
		LogLevel:  logLevel,
	}
}
