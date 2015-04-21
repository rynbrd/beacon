package main

import (
	"gopkg.in/BlueDragonX/go-settings.v0"
	"os"
	"time"
)

var (
	DefaultDockerURI       string        = "unix:///var/run/docker.sock"
	DefaultDockerPoll      time.Duration = 30 * time.Second
	DefaultEtcdURIs        []string      = []string{"http://localhost:4001/"}
	DefaultEtcdPrefix      string        = "/beacon"
	DefaultEtcdFormat      string        = "json"
	DefaultBeaconHostname  string        = getHostname()
	DefaultBeaconHeartbeat time.Duration = 30 * time.Second
	DefaultBeaconTTL       time.Duration = 30 * time.Second
	DefaultBeaconEnvVar    string        = "SERVICES"
)

func ConfigDocker(config *settings.Settings) *Docker {
	config, err := config.Object("docker")
	if err == settings.KeyError {
		config = settings.New()
	} else if err != nil {
		logger.Fatal("invalid 'docker' config object")
	}
	uri := config.StringDflt("uri", DefaultDockerURI)
	poll := config.DurationDflt("poll", DefaultDockerPoll)
	docker, err := NewDocker(uri, poll, nil)
	if err != nil {
		logger.Fatal(err.Error())
	}
	return docker
}

func ConfigEtcd(config *settings.Settings) *Etcd {
	config, err := config.Object("etcd")
	if err == settings.KeyError {
		config = settings.New()
	} else if err != nil {
		logger.Fatal("invalid 'etcd' config object")
	}

	uris := config.StringArrayDflt("uris", []string{})
	if len(uris) == 0 {
		uris = DefaultEtcdURIs
	}

	prefix := config.StringDflt("prefix", DefaultEtcdPrefix)
	format := config.StringDflt("format", DefaultEtcdFormat)
	tlsKey := config.StringDflt("tls-key", "")
	tlsCert := config.StringDflt("tls-cert", "")
	tlsCACert := config.StringDflt("tls-ca-cert", "")

	if tlsKey != "" && tlsCert != "" && tlsCACert != "" {
		for _, file := range []string{tlsKey, tlsCert, tlsCACert} {
			if !fileIsReadable(file) {
				logger.Fatalf("file '%s' is not readable", file)
			}
		}
	}

	if format != string(JSONFormat) && format != string(AddressFormat) {
		logger.Fatalf("etcd format '%s' is invalid", format)
	}

	etcd, err := NewEtcd(uris, prefix, ServiceFormat(format), tlsCert, tlsKey, tlsCACert)
	if err != nil {
		logger.Fatal(err.Error())
	}
	return etcd
}

func ConfigBeacon(config *settings.Settings) *Beacon {
	docker := ConfigDocker(config)
	etcd := ConfigEtcd(config)
	config, err := config.Object("beacon")
	if err == settings.KeyError {
		config = settings.New()
	} else if err != nil {
		logger.Fatal("invalid 'beacon' config object")
	}

	return &Beacon{
		Hostname:  config.StringDflt("hostname", DefaultBeaconHostname),
		Heartbeat: config.DurationDflt("heartbeat", DefaultBeaconHeartbeat),
		TTL:       config.DurationDflt("ttl", DefaultBeaconTTL),
		EnvVar:    config.StringDflt("env-var", DefaultBeaconEnvVar),
		Listeners: []Listener{docker},
		Discovery: etcd,
	}
}

func fileIsReadable(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	} else {
		return "localhost"
	}
}
