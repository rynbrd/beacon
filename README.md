Beacon
======
Service discovery for Docker and etcd.

How It Works
------------
Beacon announces services hosted in containers. A container publishes a service by setting an environment variable. By default this variable is `SERVICES`. You can change this with the `beacon.env-var` config file directive.

The format of the services value is `name:port/protocol`. The `name` is the name of the service to publish. The `port` is the port on which the container listens for connections. The `protocol` is either 'udp' or 'tcp'. If absent Beacon chooses 'tcp' for you. Publish more than one service by separating them with commas.

When Docker starts a container Beacon will detect this and read the services variable. Beacon looks up any port mappings at this stage. Should a mapping exist Beacon uses the hostname of the parent system and the mapped port in its announcement. Without a mapping Beacon will use the IP of the container and the port defined in the services variable. Beacon formats the final address as `hostname:port` for storage in etcd.

Beacon represents a service in etcd as a directory of keys. Each key contains an address where the service is accessible. The name of a key is the ID of the container which published that address.

The name of service directory is the name in the container's services variable. Beacon locates all service directories under a `prefix`. The prefix defaults to `/beacon`. Set the `etcd.prefix` config value to change it.

Beacon applies a TTL to all service keys. The TTL refreshes at regular intervals. If Beacon (or the entire server) dies then etcd cleans up the services when their TTL's expire.

Command Line 
------------
Beacon takes a few commandline options. These are:

`config` - The path to the config file. Defaults to /etc/beacon.yml.
`var` - The name of the service variable.
`etcd` - The etcd endpoint. May be provided multiple times.
`prefix` - The root of all etcd keys.
`docker` - The Docker endpoint.

Config File
-----------
I will lead with an example:

	# Example config.
	beacon:
	  # Refresh the services in etcd every two minutes.
	  heartbeat: 120

	  # Expire the service after 30 seconds if no heartbeat has been received.
	  ttl: 30

	docker:
	  # connect to docker here
	  uri: unix:///var/run/docker.sock

	etcd:
	  # connect to etcd here
	  uris:
	  - http://1.etcd.example.net:4001
	  - http://2.etcd.example.net:4001
	  prefix: /services

	logging:
	  # enable syslog
	  target: syslog

	  # turn on debug logging
	  level: debug

As you can see the configuration file is YAML. It consists of four appropriately named sections: `beacon`, `docker`, `etcd`, and `logging`.

### beacon
This section contains process wide configuration. The following directive are available:

* `heartbeat` - How often to refresh the etcd TTL's. Default `30s`.
* `ttl` - How long to keep a service after missing a heartbeat. Default `30s`.
* `env-var` - The name of the services variable. Default `SERVICES`.

### docker
This section configures the Docker integration.

* `uri` - Connect to Docker here. Default `unix:///var/run/docker.sock`.
* `poll` - Beacon polls Docker at this interval to retrieve missed events as a fail-safe. Default `30s`.

### etcd
This section configures the etcd integration.

* `uris` - Connect to etcd here. This is a list. Default `[ "http://localhost:4001/" ]`.
* `prefix` - The directory under which to store service directories. Default `/beacon`.
* `protocol` - If `true` then etcd will store addresses in the form `hostname:port/protocol`. Default `false`.
* `tls-key` - Path to an SSL key. Required to enable TLS.
* `tls-cert` - Path to an SSL cert. Required to enable TLS.
* `tls-ca-cert` - Path to an SSL CA cert. Required to enable TLS.

### logging
This section configures the logger.

* `target` - Log to this target. Default `stderr`.
* `level` - Only log at or above this level. Allowed values are `debug`, `info`, and `error. Default `info`.

License
-------
Copyright (c) 2015 Ryan Bourgeois. Licensed under BSD-Modified. See the [LICENSE][1] file for a copy of the license.

[1]: https://raw.githubusercontent.com/BlueDragonX/beacon/master/LICENSE "Beacon License"
