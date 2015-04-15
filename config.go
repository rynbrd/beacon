package main

import (
	"errors"
	"gopkg.in/BlueDragonX/go-settings.v0"
	"gopkg.in/BlueDragonX/simplelog.v1"
	"os"
	"strings"
	"time"
)

var (
	DefaultServiceVar       string        = "SERVICES"
	DefaultServiceTagsVar   string        = "TAGS"
	DefaultServiceHeartbeat time.Duration = 30 * time.Second
	DefaultDockerURI        string        = "unix:///var/run/docker.sock"
	DefaultEtcdURIs         []string      = []string{"http://172.17.42.1:4001/"}
	DefaultEtcdPrefix       string        = "beacon"
	DefaultEtcdTTL          time.Duration = 30 * time.Second
	DefaultTLSKey           string        = ""
	DefaultTLSCert          string        = ""
	DefaultTLSCACert        string        = ""
)

func ConfigServiceMonitor(config *settings.Settings, log *simplelog.Logger) (*ServiceMonitor, error) {
	docker := config.StringDflt("docker.uri", DefaultDockerURI)
	hostname := config.StringDflt("service.hostname", getHostname())
	tags := config.StringArrayDflt("service.tags", []string{})
	envVar := config.StringDflt("service.var", DefaultServiceVar)
	tagsVar := config.StringDflt("service.tags-var", DefaultServiceTagsVar)
	heartbeat := config.DurationDflt("service.heartbeat", DefaultServiceHeartbeat)
	return NewServiceMonitor(docker, hostname, tags, envVar, tagsVar, heartbeat, log)
}

func ConfigServiceAnnouncer(config *settings.Settings, log *simplelog.Logger) (*ServiceAnnouncer, error) {
	uris := config.StringArrayDflt("uris", []string{})
	if len(uris) == 0 {
		uris = DefaultEtcdURIs
	}
	for n, uri := range uris {
		uris[n] = strings.TrimRight(uri, "/")
	}

	tlsKey := config.StringDflt("etcd.tls-key", DefaultTLSKey)
	tlsCert := config.StringDflt("etcd.tls-cert", DefaultTLSCert)
	tlsCACert := config.StringDflt("etcd.tls-ca-cert", DefaultTLSCACert)
	ttl := config.DurationDflt("etcd.ttl", DefaultEtcdTTL)
	heartbeat := config.DurationDflt("service.heartbeat", DefaultServiceHeartbeat)
	prefix := config.StringDflt("etcd.prefix", DefaultEtcdPrefix)
	ttlSeconds := uint64((ttl + heartbeat).Seconds())

	if tlsKey != "" && tlsCert != "" && tlsCACert != "" {
		if !fileIsReadable(tlsKey) {
			return nil, errors.New("invalid etcd.tls-key: file is not readable")
		}
		if !fileIsReadable(tlsCert) {
			return nil, errors.New("invalid etcd.tls-cert: file is not readable")
		}
		if !fileIsReadable(tlsCACert) {
			return nil, errors.New("invalid etcd.tls-ca-cert: file is not readable")
		}
		return NewTLSServiceAnnouncer(uris, tlsCert, tlsKey, tlsCACert, prefix, ttlSeconds, log)
	} else {
		ann := NewServiceAnnouncer(uris, prefix, ttlSeconds, log)
		return ann, nil
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
