package main

import (
	"errors"
	"fmt"
	"github.com/BlueDragonX/go-etcd/etcd"
	"gopkg.in/BlueDragonX/simplelog.v1"
	"strconv"
	"strings"
)

// Return true if the err is an EtcdError and has the given error code.
func checkEtcdErrorCode(err error, code int) bool {
	etcdErr, ok := err.(*etcd.EtcdError)
	if ok && etcdErr.ErrorCode == code {
		return true
	}
	return false
}

type ServiceAnnouncer struct {
	client *etcd.Client
	prefix string
	ttl    uint64
	log    *simplelog.Logger
}

func newServiceAnnouncer(client *etcd.Client, prefix string, ttl uint64, log *simplelog.Logger) *ServiceAnnouncer {
	if prefix != "" {
		if prefix[0] != '/' {
			prefix = "/" + prefix
		}
		if prefix[len(prefix)-1] == '/' {
			prefix = prefix[:len(prefix)-1]
		}
	}
	ann := &ServiceAnnouncer{}
	ann.client = client
	ann.prefix = prefix
	ann.ttl = ttl
	ann.log = log
	return ann
}

// Create a new service announcer. Announce new services to the given etcd cluster.
func NewServiceAnnouncer(urls []string, prefix string, ttl uint64, log *simplelog.Logger) *ServiceAnnouncer {
	cleanUrls := make([]string, len(urls))
	for i, url := range urls {
		cleanUrls[i] = strings.TrimRight(url, "/")
	}
	return newServiceAnnouncer(etcd.NewClient(cleanUrls), prefix, ttl, log)
}

// Create a new service announcer. Announce new services to the given etcd cluster over TLS.
func NewTLSServiceAnnouncer(urls []string, cert, key, caCert, prefix string, ttl uint64, log *simplelog.Logger) (ann *ServiceAnnouncer, err error) {
	var client *etcd.Client
	if client, err = etcd.NewTLSClient(urls, cert, key, caCert); err == nil {
		ann = newServiceAnnouncer(client, prefix, ttl, log)
	}
	return
}

// Increment an index counter in etcd. Replace or create the key if it's not a valid int.
func (ann *ServiceAnnouncer) increment(key string) (value int, err error) {

	createKey := func() (value int, err error) {
		_, err = ann.client.Set(key, "1", 0)
		return 1, err
	}

	replaceKey := func() (value int, err error) {
		if _, err = ann.client.Delete(key, true); err == nil {
			value, err = createKey()
		}
		return
	}

	incrementKey := func(value int) (int, error) {
		newValue := value + 1
		newValueStr := fmt.Sprintf("%v", newValue)
		oldValueStr := fmt.Sprintf("%v", value)
		_, err := ann.client.CompareAndSwap(key, newValueStr, 0, oldValueStr, 0)
		return newValue, err
	}

	var response *etcd.Response

	for {
		if response, err = ann.client.Get(key, false, false); err != nil {
			value, err = createKey()
		} else if response.Node.Dir {
			// replace the directory with zero
			value, err = replaceKey()
		} else {
			if value, err = strconv.Atoi(response.Node.Value); err != nil {
				// parse the value and replace it with zero if we can't
				value, err = replaceKey()
			} else {
				// increment the value
				value, err = incrementKey(value)
				if err != nil {
					// convert value
					if etcdError, ok := err.(*etcd.EtcdError); ok {
						if etcdError.ErrorCode == 101 {
							continue
						}
					}
				}
			}
		}
		break
	}

	if err == nil {
		ann.log.Debug("increment %s to %d", key, value)
	} else {
		ann.log.Error("failed to increment %s: %s", key, err)
	}
	return
}

// Increment all indexes.
func (ann *ServiceAnnouncer) incrementIndexes(svc *Service) error {
	indexes := []string{
		fmt.Sprintf("%v/_index", ann.prefix),
		fmt.Sprintf("%v/%v/_index", ann.prefix, svc.Name),
	}

	for _, index := range indexes {
		if _, err := ann.increment(index); err != nil {
			return err
		}
	}
	return nil
}

// Return the path to the directory for a service.
func (ann *ServiceAnnouncer) getServicePath(svc *Service) string {
	return fmt.Sprintf("%v/%v/%v", ann.prefix, svc.Name, svc.ContainerId)
}

func (ann *ServiceAnnouncer) setValue(svc *Service, root, name, value string) (err error) {
	key := fmt.Sprintf("%v/%v", root, name)
	_, err = ann.client.Set(key, value, 0)
	if err == nil {
		ann.log.Debug("etcd set '%s' = '%s'", key, value)
	} else {
		ann.log.Error("etcd failed to set '%s' = '%s'", key, value)
	}
	return
}

func (ann *ServiceAnnouncer) addService(svc *Service) (err error) {
	root := ann.getServicePath(svc)
	if _, err = ann.client.SetDir(root, ann.ttl); err != nil {
		if checkEtcdErrorCode(err, 102) {
			_, err = ann.client.UpdateDir(root, ann.ttl)
		}
		if err != nil {
			return
		}
	}
	if err = ann.setValue(svc, root, "container-name", svc.ContainerName); err != nil {
		return
	}
	if err = ann.setValue(svc, root, "container-port", fmt.Sprintf("%d", svc.ContainerPort)); err != nil {
		return
	}
	if err = ann.setValue(svc, root, "host-name", svc.HostName); err != nil {
		return
	}
	if err = ann.setValue(svc, root, "host-port", fmt.Sprintf("%d", svc.HostPort)); err != nil {
		return
	}
	if err = ann.setValue(svc, root, "protocol", svc.Protocol); err != nil {
		return
	}
	if err = ann.incrementIndexes(svc); err != nil {
		return
	}
	return
}

func (ann *ServiceAnnouncer) heartbeatService(svc *Service) (err error) {
	root := ann.getServicePath(svc)
	_, err = ann.client.UpdateDir(root, ann.ttl)
	if checkEtcdErrorCode(err, 100) {
		err = ann.addService(svc)
	}
	return
}

func (ann *ServiceAnnouncer) cleanService(svc *Service) (err error) {
	svcKey := fmt.Sprintf("%v/%v", ann.prefix, svc.Name)
	indexKey := fmt.Sprintf("%v/index", svcKey)

	var response *etcd.Response
	if response, err = ann.client.Get(svcKey, false, false); err != nil {
		return
	}
	if !response.Node.Dir || len(response.Node.Nodes) > 1 {
		return
	}

	var indexNode *etcd.Node
	for _, node := range response.Node.Nodes {
		if node.Key == indexKey {
			indexNode = node
			break
		}
	}

	if indexNode == nil {
		return errors.New(fmt.Sprintf("%s not found", indexKey))
	}

	var index int
	if index, err = strconv.Atoi(indexNode.Value); err != nil {
		return
	}
	if _, err = ann.client.CompareAndDelete(indexKey, fmt.Sprintf("%d", index), 0); err != nil {
		return
	}
	_, err = ann.client.DeleteDir(svcKey)
	return
}

func (ann *ServiceAnnouncer) removeService(svc *Service) (err error) {
	root := ann.getServicePath(svc)
	key := fmt.Sprintf("%v/%v", ann.prefix, svc.Name)
	_, err = ann.client.Delete(root, true)
	if err == nil {
		err = ann.incrementIndexes(svc)
		if err == nil {
			err = ann.cleanService(svc)
		}
	}
	if err == nil {
		ann.log.Debug("etcd delete '%s'", key)
	} else {
		ann.log.Debug("etcd delete '%s' failed: %s", key, err)
	}
	return
}

func (ann *ServiceAnnouncer) Announce(event ServiceEvent) (err error) {
	switch event.State {
	case Add:
		err = ann.addService(event.Service)
	case Heartbeat:
		err = ann.heartbeatService(event.Service)
	case Remove:
		err = ann.removeService(event.Service)
	}
	return
}
