package main

import (
	"errors"
	"flag"
	"github.com/BlueDragonX/yamlcfg"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"time"
)

const (
	DefaultSyslog     = false
	DefaultConsole    = true
	DefaultConfigFile = "config.yml"
	DefaultDockerURL  = "unix:///var/run/docker.sock"
	DefaultDockerVar  = "SERVICES"
	DefaultEtcdURL    = "http://172.17.42.1:4001/"
	DefaultEtcdPrefix = "services"
	DefaultHostname   = ""
	DefaultHeartbeat  = 30
	DefaultTTL        = 30
	DefaultTLSKey     = ""
	DefaultTLSCert    = ""
	DefaultTLSCACert  = ""
)

func fileIsReadable(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type Config struct {
	Syslog     bool
	Console    bool
	DockerURL  string
	DockerVar  string
	EtcdURLs   []string
	EtcdPrefix string
	Hostname   string
	Heartbeat  time.Duration
	TTL        time.Duration
	TLSKey     string
	TLSCert    string
	TLSCACert  string
}

// Create a new config struct with default values.
func NewConfig() Config {
	hostname := DefaultHostname
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	return Config{
		DefaultSyslog,
		DefaultConsole,
		DefaultDockerURL,
		DefaultDockerVar,
		[]string{DefaultEtcdURL},
		DefaultEtcdPrefix,
		hostname,
		DefaultHeartbeat * time.Second,
		DefaultTTL * time.Second,
		DefaultTLSKey,
		DefaultTLSCert,
		DefaultTLSCACert,
	}
}

// Check if TLS is enabled.
func (cfg Config) IsTLS() bool {
	return cfg.TLSKey != "" && cfg.TLSCert != "" && cfg.TLSCACert != ""
}

// SetYAML parses the YAML tree into the object.
func (cfg *Config) SetYAML(tag string, data interface{}) bool {
	config.AssertIsMap("config", data)
	cfg.Syslog = config.GetBool(data, "syslog", DefaultSyslog)
	cfg.Console = config.GetBool(data, "console", DefaultConsole)
	cfg.DockerURL = config.GetString(data, "docker-url", DefaultDockerURL)
	cfg.DockerVar = config.GetString(data, "docker-var", DefaultDockerVar)
	cfg.EtcdURLs = config.GetStringArray(data, "etcd-urls", []string{DefaultEtcdURL})
	cfg.EtcdPrefix = config.GetString(data, "etcd-prefix", DefaultEtcdPrefix)
	cfg.Hostname = config.GetString(data, "hostname", DefaultHostname)
	cfg.Heartbeat = config.GetDuration(data, "heartbeat", DefaultHeartbeat*time.Second)
	cfg.TTL = config.GetDuration(data, "ttl", DefaultTTL*time.Second)
	cfg.TLSKey = config.GetString(data, "tls-key", DefaultTLSKey)
	cfg.TLSCert = config.GetString(data, "tls-cert", DefaultTLSCert)
	cfg.TLSCACert = config.GetString(data, "tls-ca-cert", DefaultTLSCACert)
	if cfg.Hostname == "" {
		cfg.Hostname, _ = os.Hostname()
	}
	return true
}

// Validate the config.
func (cfg Config) Validate() error {
	if cfg.DockerURL == "" {
		return errors.New("invalid value for Docker URL")
	}
	if cfg.DockerVar == "" {
		return errors.New("invalid value for Docker Var")
	}
	if len(cfg.EtcdURLs) == 0 || cfg.EtcdURLs[0] == "" {
		return errors.New("invalid value for etcd URLs")
	}
	if cfg.EtcdPrefix == "" {
		return errors.New("invalid value for etcd prefix")
	}
	if cfg.Hostname == "" {
		return errors.New("invalid value for hostname")
	}
	if cfg.Heartbeat <= 0*time.Second {
		return errors.New("invalid value for heartbeat")
	}
	if cfg.TTL <= 0*time.Second {
		return errors.New("invalid value for TTL")
	}
	if cfg.IsTLS() {
		if !fileIsReadable(cfg.TLSKey) {
			return errors.New("TLS key is not readable")
		}
		if !fileIsReadable(cfg.TLSCert) {
			return errors.New("TLS cert is not readable")
		}
		if !fileIsReadable(cfg.TLSCACert) {
			return errors.New("TLS CA cert is not readable")
		}
	}
	return nil
}

// Load the Deckhand configuration file.
func LoadConfig() (cfg Config, err error) {
	var file string
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&file, "config", DefaultConfigFile, "YAML configuration file")
	flags.Parse(os.Args[1:])

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	err = goyaml.Unmarshal(data, &cfg)
	return
}
