package main

import (
	"flag"
	"strings"
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
	Config string
	Var    string
	Etcd   []string
	Docker string
	Prefix string
}

func ParseOptionsOrExit(args []string) *Options {
	var config string
	var envVar string
	var etcd stringsOpt
	var docker string
	var prefix string

	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.StringVar(&config, "config", DefaultConfigFile, "The path to the config file.")
	flags.StringVar(&envVar, "var", DefaultConfigFile, "The container environment variable which holds the hosted services value.")
	flags.Var(&etcd, "etcd", "The etcd endpoint. May be provided multiple times.")
	flags.StringVar(&docker, "docker", "", "The Docker endpoint.")
	flags.StringVar(&prefix, "prefix", "", "A prefix to prepend to all etcd key paths.")
	flags.Parse(args[1:])

	return &Options{
		Config: config,
		Var:    envVar,
		Etcd:   []string(etcd),
		Docker: docker,
		Prefix: prefix,
	}
}
