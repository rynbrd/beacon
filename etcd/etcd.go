package etcd

import (
	"encoding/json"
	"github.com/BlueDragonX/beacon/container"
	"github.com/coreos/go-etcd/etcd"
	"math"
	"strings"
	"time"
)

type ServiceFormat string

const (
	JSONFormat    ServiceFormat = "json"
	AddressFormat               = "address"
)

// Etcd announces services into etcd.
type Etcd struct {
	client  *etcd.Client
	prefix  string
	format  ServiceFormat
	stopped chan struct{}
}

// NewEtcd creates a new Etcd discovery backend. TLS is enabled if all TLS
// parameters are provided.
func NewEtcd(uris []string, prefix string, format ServiceFormat, tlsCert, tlsKey, tlsCACert string) (*Etcd, error) {
	var err error
	var client *etcd.Client
	for n, uri := range uris {
		uris[n] = strings.TrimRight(uri, "/")
	}
	if tlsCert != "" && tlsKey != "" && tlsCACert != "" {
		client, err = etcd.NewTLSClient(uris, tlsCert, tlsKey, tlsCACert)
		if err != nil {
			return nil, err
		}
	} else {
		client = etcd.NewClient(uris)
	}

	prefix = strings.Trim(prefix, "/")
	etcd := &Etcd{client, prefix, format, make(chan struct{})}
	go etcd.cleanup()
	return etcd, nil
}

// Announce a service.
func (e *Etcd) Announce(name, container string, address *container.Address, ttl time.Duration) error {
	var err error
	var value string
	path := e.joinPath(e.prefix, name, container)
	if value, err = e.formatAddress(address); err == nil {
		ttlSecs := ttl.Seconds()
		ttlInt := uint64(ttlSecs + math.Copysign(0.5, ttlSecs))
		if _, err = e.client.Set(path, value, ttlInt); err == nil {
			logger.Printf("etcd set of '%s=%s' successful", path, value)
		} else {
			logger.Printf("etcd set of '%s=%s' failed: %s", path, value, err)
		}
	}
	return err
}

// Shutdown a service.
func (e *Etcd) Shutdown(name, container string) error {
	path := e.joinPath(e.prefix, name, container)
	_, err := e.client.Delete(path, false)
	if err == nil {
		logger.Printf("etcd rm of '%s' successful", path)
	} else {
		logger.Printf("etcd rm of '%s' failed: %s", path, err)
	}
	return err
}

// Close the etcd service.
func (e *Etcd) Close() error {
	e.client.Close()
	return nil
}

// cleanup removes empty service directories every five minutes.
func (e *Etcd) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			logger.Print("etcd cleanup started")
			path := e.joinPath(e.prefix)
			res, err := e.client.Get(path, false, false)
			if err != nil {
				logger.Printf("etcd ls of %s failed: %s", e.prefix, err)
				continue
			}
			if !res.Node.Dir {
				continue
			}
			for _, node := range res.Node.Nodes {
				if node.Dir {
					e.rm(node.Key)
				}
			}
			logger.Print("etcd cleanup complete")
		case <-e.stopped:
			return
		}
	}
}

// rm removes a key or empty directory.
func (e *Etcd) rm(path string) {
	if _, err := e.client.DeleteDir(path); err == nil {
		logger.Printf("etcd rm of %s successful", path)
	} else {
		logger.Printf("etcd rm of %s failed: %s", path, err)
	}
}

// joinPath joins etcd paths.
func (e *Etcd) joinPath(args ...string) string {
	path := ""
	for _, arg := range args {
		arg = strings.Trim(arg, "/")
		if arg != "" {
			path += "/" + arg
		}
	}
	return path
}

// format an address for storage
func (e *Etcd) formatAddress(addr *container.Address) (string, error) {
	switch e.format {
	case JSONFormat:
		data := struct {
			Host     string
			Port     int
			Protocol string
		}{addr.Hostname, addr.Port.Number, addr.Port.Protocol}
		if bytes, err := json.Marshal(data); err == nil {
			return string(bytes), nil
		} else {
			return "", err
		}
	default:
		return addr.StringNoProtocol(), nil
	}
}
