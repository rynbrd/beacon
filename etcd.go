package main

import (
	"github.com/coreos/go-etcd/etcd"
	"strings"
	"time"
)

// Etcd announces services into etcd.
type Etcd struct {
	client   *etcd.Client
	prefix   string
	protocol bool
	stopped  chan struct{}
}

// NewEtcd creates a new Etcd discovery backend. TLS is enabled if all TLS
// parameters are provided.
func NewEtcd(uris []string, prefix string, protocol bool, tlsCert, tlsKey, tlsCACert string) (*Etcd, error) {
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
	etcd := &Etcd{client, prefix, protocol, make(chan struct{})}
	go etcd.cleanup()
	return etcd, nil
}

// Announce a service.
func (e *Etcd) Announce(name, container string, address *Address, ttl time.Duration) error {
	path := e.joinPath(e.prefix, name, container)
	var value string
	if e.protocol {
		value = address.String()
	} else {
		value = address.StringNoProtocol()
	}
	_, err := e.client.Set(path, value, 0)
	if err == nil {
		logger.Debugf("etcd set of '%s=%s' successful", path, value)
	} else {
		logger.Errorf("etcd set of '%s=%s' failed: %s", path, value, err)
	}
	return err
}

// Shutdown a service.
func (e *Etcd) Shutdown(name, container string) error {
	path := e.joinPath(e.prefix, name, container)
	_, err := e.client.Delete(path, false)
	if err == nil {
		logger.Debugf("etcd rm of '%s' successful", path)
	} else {
		logger.Errorf("etcd rm of '%s' failed: %s", path, err)
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
			logger.Debug("etcd cleanup started")
			path := e.joinPath(e.prefix)
			res, err := e.client.Get(path, false, false)
			if err != nil {
				logger.Errorf("etcd ls of %s failed: %s", e.prefix, err)
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
			logger.Debug("etcd cleanup complete")
		case <-e.stopped:
			return
		}
	}
}

// rm removes a key or empty directory.
func (e *Etcd) rm(path string) {
	if _, err := e.client.DeleteDir(path); err == nil {
		logger.Debugf("etcd rm of %s successful", path)
	} else {
		logger.Debugf("etcd rm of %s failed: %s", path, err)
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
