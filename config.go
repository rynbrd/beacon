package main

import (
	"errors"
	"flag"
	"gopkg.in/BlueDragonX/simplelog.v1"
	"gopkg.in/BlueDragonX/yamlcfg.v1"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"time"
)

const (
	DefaultConfigFile       = "config.yml"
	DefaultLoggingSyslog    = false
	DefaultLoggingConsole   = true
	DefaultLoggingLevel     = simplelog.NOTICE
	DefaultServiceVar       = "SERVICES"
	DefaultServiceHeartbeat = 30
	DefaultServiceTtl       = 30
	DefaultDockerURI        = "unix:///var/run/docker.sock"
	DefaultEtcdURI          = "http://172.17.42.1:4001/"
	DefaultEtcdPrefix       = "beacon"
	DefaultEtcdTLSKey       = ""
	DefaultEtcdTLSCert      = ""
	DefaultEtcdTLSCACert    = ""
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
	Level   int
}

// Get default logging config.
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		DefaultLoggingSyslog,
		DefaultLoggingConsole,
		DefaultLoggingLevel,
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *LoggingConfig) SetYAML(tag string, data interface{}) bool {
	yamlcfg.AssertIsMap("logging", data)
	cfg.Syslog = yamlcfg.GetBool(data, "syslog", DefaultLoggingSyslog)
	cfg.Console = yamlcfg.GetBool(data, "console", DefaultLoggingConsole)
	levelStr := yamlcfg.GetString(data, "level", "")
	if levelStr == "" {
		cfg.Level = DefaultLoggingLevel
	} else {
		cfg.Level = simplelog.StringToLevel(levelStr)
	}
	return true
}

// Store service related configuration.
type ServiceConfig struct {
	Var       string
	Hostname  string
	Heartbeat time.Duration
	Ttl       time.Duration
}

// Get default service config.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		DefaultServiceVar,
		getHostname(""),
		DefaultServiceHeartbeat * time.Second,
		DefaultServiceTtl,
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *ServiceConfig) SetYAML(tag string, data interface{}) bool {
	yamlcfg.AssertIsMap("service", data)
	cfg.Var = yamlcfg.GetString(data, "var", DefaultServiceVar)
	cfg.Hostname = getHostname(yamlcfg.GetString(data, "hostname", ""))
	cfg.Heartbeat = yamlcfg.GetDuration(data, "heartbeat", DefaultServiceHeartbeat*time.Second)
	cfg.Ttl = yamlcfg.GetDuration(data, "ttl", DefaultServiceTtl*time.Second)
	return true
}

// Validate the configuration.
func (cfg *ServiceConfig) Validate() []error {
	errs := []error{}
	if cfg.Var == "" {
		errs = append(errs, errors.New("invalid value for service.var"))
	}
	if cfg.Hostname == "" {
		errs = append(errs, errors.New("invalid value for service.hostname"))
	}
	if cfg.Heartbeat <= 0*time.Second {
		errs = append(errs, errors.New("invalid value for service.heartbeat"))
	}
	if cfg.Ttl <= 0*time.Second {
		errs = append(errs, errors.New("invalid value for service.ttl"))
	}
	return errs
}

// Store Docker related configuration.
type DockerConfig struct {
	URI string
}

// Get default Docker config.
func DefaultDockerConfig() DockerConfig {
	return DockerConfig{
		DefaultDockerURI,
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *DockerConfig) SetYAML(tag string, data interface{}) bool {
	yamlcfg.AssertIsMap("docker", data)
	cfg.URI = yamlcfg.GetString(data, "uri", DefaultDockerURI)
	return true
}

// Store etcd related configuration.
type EtcdConfig struct {
	URIs      []string
	Prefix    string
	TLSKey    string
	TLSCert   string
	TLSCACert string
}

//Get default etcd config.
func DefaultEtcdConfig() EtcdConfig {
	return EtcdConfig{
		[]string{DefaultEtcdURI},
		DefaultEtcdPrefix,
		DefaultEtcdTLSKey,
		DefaultEtcdTLSCert,
		DefaultEtcdTLSCACert,
	}
}

// SetYAML parses the YAML tree into the object.
func (cfg *EtcdConfig) SetYAML(tag string, data interface{}) bool {
	yamlcfg.AssertIsMap("etcd", data)
	cfg.URIs = yamlcfg.GetStringArray(data, "uris", []string{})

	uri := yamlcfg.GetString(data, "uri", "")
	if uri != "" {
		cfg.URIs = append(cfg.URIs, uri)
	}
	if len(cfg.URIs) == 0 {
		cfg.URIs = append(cfg.URIs, DefaultEtcdURI)
	}

	cfg.Prefix = yamlcfg.GetString(data, "prefix", DefaultEtcdPrefix)
	cfg.TLSKey = yamlcfg.GetString(data, "tls-key", DefaultEtcdTLSKey)
	cfg.TLSCert = yamlcfg.GetString(data, "tls-cert", DefaultEtcdTLSCert)
	cfg.TLSCACert = yamlcfg.GetString(data, "tls-ca-cert", DefaultEtcdTLSCACert)
	return true
}

// Check if TLS is enabled.
func (cfg EtcdConfig) IsTLS() bool {
	return cfg.TLSKey != "" && cfg.TLSCert != "" && cfg.TLSCACert != ""
}

// Root config object.
type Config struct {
	Logging LoggingConfig
	Service ServiceConfig
	Docker  DockerConfig
	Etcd    EtcdConfig
}

// SetYAML parses the YAML tree into the object.
func (cfg *Config) SetYAML(tag string, data interface{}) bool {
	yamlcfg.AssertIsMap("config", data)
	if loggingData, ok := yamlcfg.GetMapItem(data, "logging"); ok {
		cfg.Logging.SetYAML("logging", loggingData)
	} else {
		cfg.Logging = DefaultLoggingConfig()
	}

	if serviceData, ok := yamlcfg.GetMapItem(data, "service"); ok {
		cfg.Service.SetYAML("service", serviceData)
	} else {
		cfg.Service = DefaultServiceConfig()
	}

	if dockerData, ok := yamlcfg.GetMapItem(data, "docker"); ok {
		cfg.Docker.SetYAML("docker", dockerData)
	} else {
		cfg.Docker = DefaultDockerConfig()
	}

	if etcdData, ok := yamlcfg.GetMapItem(data, "etcd"); ok {
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
	if cfg.URI == "" {
		errs = append(errs, errors.New("invalid value for docker.uri"))
	}
	return errs
}

// Validate the configuration.
func (cfg *EtcdConfig) Validate() []error {
	errs := []error{}
	if len(cfg.URIs) == 0 || cfg.URIs[0] == "" {
		errs = append(errs, errors.New("invalid value for etcd.uris"))
	}
	if cfg.Prefix == "" {
		errs = append(errs, errors.New("invalid value for etcd.prefix"))
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
	errs = append(errs, cfg.Service.Validate()...)
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
	err = yaml.Unmarshal(data, &cfg)
	return
}
