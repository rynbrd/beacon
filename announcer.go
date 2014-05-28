package main

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
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
}

func newServiceAnnouncer(client *etcd.Client, prefix string, ttl uint64) *ServiceAnnouncer {
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
	return ann
}

// Create a new service announcer. Announce new services to the given etcd cluster.
func NewServiceAnnouncer(urls []string, prefix string, ttl uint64) *ServiceAnnouncer {
	return newServiceAnnouncer(etcd.NewClient(urls), prefix, ttl)
}

// Create a new service announcer. Announce new services to the given etcd cluster over TLS.
func NewTLSServiceAnnouncer(urls []string, cert, key, caCert, prefix string, ttl uint64) (ann *ServiceAnnouncer, err error) {
	var client *etcd.Client
	if client, err = etcd.NewTLSClient(urls, cert, key, caCert); err == nil {
		ann = newServiceAnnouncer(client, prefix, ttl)
	}
	return
}

// Return the path to the directory for a service.
func (ann *ServiceAnnouncer) getServicePath(svc *Service) string {
	return fmt.Sprintf("%v/%v/%v", ann.prefix, svc.Name, svc.ContainerId)
}

func (ann *ServiceAnnouncer) setValue(svc *Service, root, name, value string) (err error) {
	_, err = ann.client.Set(fmt.Sprintf("%v/%v", root, name), value, 0)
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

func (ann *ServiceAnnouncer) removeService(svc *Service) (err error) {
	root := ann.getServicePath(svc)
	_, err = ann.client.Delete(root, true)
	ann.client.DeleteDir(fmt.Sprintf("%v/%v", ann.prefix, svc.Name))
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
