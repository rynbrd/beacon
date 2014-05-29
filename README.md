Beacon
======
Service discovery for Docker and etcd!

Installing
----------
You can use the go command to get and install the package:

    go get github.com/BlueDragonX/beacon
    go install github.com/BlueDragonX/beacon

Configuring
-----------
The redflag binary takes one option: -config. This takes as an argument the
path to the config file to load. This defaults to "config.yml".

The config file is a YAML file. It may contain the following options:

- *syslog* - Control logging to syslog. Boolean. Defaults to false.
- *console* - Control logging to console. Boolean. Defaults to true.
- *docker-url* - The path to the Docker socket. String. Defaults to
  "unix://var/run/docker.sock".
- *docker-var* - The environment variable published by the Docker container
  which defines services exposed by the container. String. Defaults to
  "SERVICES".
- *etcd-urls* - A list of etcd URL's to connect to. Array of strings. Defaults
  to ["http://172.17.42.1:4001/"].
- *etcd-prefix* - The root directory in which to store discovered services in.
  String. Defaults to "services".
- *hostname* - The DNS name of the Docker host. This is the hostname stored in
  etcd where external services can access discovered services. String. Defaults
  to the output of hostname.
- *heartbeat* - How often (in seconds) services will be refreshed in etcd.
  Integer. Defaults to 30.
- *ttl* - How long (in seconds) a service should remain active after receiving
  a heartbeat. Integer. Defaults to 30.
- *tls-key* - The TLS key to use when communicating with etcd. String. Defaults
  to "".
- *tls-cert* - The TLS cert to use when communicating with etcd. String.
  Defaults to "".
- *tls-ca-cert* - The TLS CA cert to use when communicating with etcd. String.
  Defaults to "".

Services
--------
Services are announced to etcd under the _etcd-prefix_ defined in the config file. The service paths are structured as follows:

    /<service_name>/<container_id>/<keys>

The following keys are set fore each service/container:

- *container-name* - The hostname of the container the service is running on.
- *container-port* - The port the container exposes for this service.
- *host-name* - The DNS name (or possibly an IP address) of the host running the container.
- *host-port* - The port on the host that is mapped to this service.
- *protocol* - The protocol the service. Either tcp or udp.

License
-------
Copyright (c) 2014 Ryan Bourgeois. Licensed under BSD-Modified. See the LICENSE
file for a copy of the license.
