package main

import (
	"errors"
	"flag"
	h "github.com/BlueDragonX/yamlcfg"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"time"
)

const (
	DefaultConfigFile     = "config.yml"
	DefaultLoggingSyslog  = false
	DefaultLoggingConsole = true
	DefaultDockerURL      = "unix:///var/run/docker.sock"
	DefaultDockerVar      = "SERVICES"
	DefaultEtcdURL        = "http://172.17.42.1:4001/"
	DefaultEtcdPrefix     = "services"
	DefaultEtcdHeartbeat  = 30
	DefaultEtcdTTL        = 30
	DefaultEtcdTLSKey     = ""
	DefaultEtcdTLSCert    = ""
	DefaultEtcdTLSCACert  = ""
)

func fileIsReadable(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getHostname(hostname string) string {
	if hostname == "" {
		var err error
		if hostname, err = os.Hostname(); err != nil {
			hostname = "localhost"
		}
	}
	return hostname
}

// Store log related configuration.
type LoggingConfig struct {
	Syslog  bool
	Console bool
}

// Get default logging config.
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		DefaultLoggingSyslog,
		DefaultLoggingConsole,
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *LoggingConfig) SetYAML(tag string, data interface{}) bool {
	h.AssertIsMap("logging", data)
	cfg.Syslog = h.GetBool(data, "syslog", DefaultLoggingSyslog)
	cfg.Console = h.GetBool(data, "console", DefaultLoggingConsole)
	return true
}

// Store Docker related configuration.
type DockerConfig struct {
	URL      string
	Var      string
	Hostname string
}

// Get default Docker config.
func DefaultDockerConfig() DockerConfig {
	return DockerConfig{
		DefaultDockerURL,
		DefaultDockerVar,
		getHostname(""),
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *DockerConfig) SetYAML(tag string, data interface{}) bool {
	h.AssertIsMap("docker", data)
	cfg.URL = h.GetString(data, "url", DefaultDockerURL)
	cfg.Var = h.GetString(data, "var", DefaultDockerVar)
	cfg.Hostname = getHostname(h.GetString(data, "hostname", ""))
	return true
}

// Store etcd related configuration.
type EtcdConfig struct {
	URLs      []string
	Prefix    string
	Heartbeat time.Duration
	TTL       time.Duration
	TLSKey    string
	TLSCert   string
	TLSCACert string
}

//Get default etcd config.
func DefaultEtcdConfig() EtcdConfig {
	return EtcdConfig{
		[]string{DefaultEtcdURL},
		DefaultEtcdPrefix,
		DefaultEtcdHeartbeat * time.Second,
		DefaultEtcdTTL,
		DefaultEtcdTLSKey,
		DefaultEtcdTLSCert,
		DefaultEtcdTLSCACert,
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *EtcdConfig) SetYAML(tag string, data interface{}) bool {
	h.AssertIsMap("etcd", data)
	cfg.URLs = h.GetStringArray(data, "urls", []string{DefaultEtcdURL})
	cfg.Prefix = h.GetString(data, "prefix", DefaultEtcdPrefix)
	cfg.Heartbeat = h.GetDuration(data, "heartbeat", DefaultEtcdHeartbeat*time.Second)
	cfg.TTL = h.GetDuration(data, "ttl", DefaultEtcdTTL*time.Second)
	cfg.TLSKey = h.GetString(data, "tls-key", DefaultEtcdTLSKey)
	cfg.TLSCert = h.GetString(data, "tls-cert", DefaultEtcdTLSCert)
	cfg.TLSCACert = h.GetString(data, "tls-ca-cert", DefaultEtcdTLSCACert)
	return true
}

// Check if TLS is enabled.
func (cfg EtcdConfig) IsTLS() bool {
	return cfg.TLSKey != "" && cfg.TLSCert != "" && cfg.TLSCACert != ""
}

// Root config object.
type Config struct {
	Logging LoggingConfig
	Docker  DockerConfig
	Etcd    EtcdConfig
}

// SetYAML parses the YAML tree into the object.
func (cfg *Config) SetYAML(tag string, data interface{}) bool {
	h.AssertIsMap("config", data)
	if loggingData, ok := h.GetMapItem(data, "logging"); ok {
		cfg.Logging.SetYAML("logging", loggingData)
	} else {
		cfg.Logging = DefaultLoggingConfig()
	}

	if dockerData, ok := h.GetMapItem(data, "docker"); ok {
		cfg.Docker.SetYAML("docker", dockerData)
	} else {
		cfg.Docker = DefaultDockerConfig()
	}

	if etcdData, ok := h.GetMapItem(data, "etcd"); ok {
		cfg.Etcd.SetYAML("etcd", etcdData)
	} else {
		cfg.Etcd = DefaultEtcdConfig()
	}
	return true
}

// Validate the configuration.
func (cfg *LoggingConfig) Validate() []error {
	return []error{}
}

// Validate the configuration.
func (cfg *DockerConfig) Validate() []error {
	errs := []error{}
	if cfg.URL == "" {
		errs = append(errs, errors.New("invalid value for docker.url"))
	}
	if cfg.Var == "" {
		errs = append(errs, errors.New("invalid value for docker.var"))
	}
	if cfg.Hostname == "" {
		errs = append(errs, errors.New("invalid value for docker.hostname"))
	}
	return errs
}

// Validate the configuration.
func (cfg *EtcdConfig) Validate() []error {
	errs := []error{}
	if len(cfg.URLs) == 0 || cfg.URLs[0] == "" {
		errs = append(errs, errors.New("invalid value for etcd.urls"))
	}
	if cfg.Prefix == "" {
		errs = append(errs, errors.New("invalid value for etcd. prefix"))
	}
	if cfg.Heartbeat <= 0*time.Second {
		errs = append(errs, errors.New("invalid value for etcd.heartbeat"))
	}
	if cfg.TTL <= 0*time.Second {
		errs = append(errs, errors.New("invalid value for etcd.ttl"))
	}
	if cfg.IsTLS() {
		if !fileIsReadable(cfg.TLSKey) {
			errs = append(errs, errors.New("invalid etcd.tls-key: file is not readable"))
		}
		if !fileIsReadable(cfg.TLSCert) {
			errs = append(errs, errors.New("invalid etcd.tls-cert: file is not readable"))
		}
		if !fileIsReadable(cfg.TLSCACert) {
			errs = append(errs, errors.New("invalid etcd.tls-ca-cert: file is not readable"))
		}
	}
	return errs
}

// Validate the config.
func (cfg Config) Validate() []error {
	errs := []error{}
	errs = append(errs, cfg.Logging.Validate()...)
	errs = append(errs, cfg.Docker.Validate()...)
	errs = append(errs, cfg.Etcd.Validate()...)
	return errs
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
