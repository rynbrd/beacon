Beacon
======
Service discovery for Docker and etcd!

How It Works
------------
Beacon listens for `start` and `die` events in Docker. When a `start` event is
recieved Beacon will read the service configuration on the running container,
look up port mappings for the defined available services, and place this
information into etcd. On `die` Beacon will remove the items it added to etcd
for that container.

Beacon will poll Docker on start to add services to etcd. It will also remove
any announced services on shutdown. Beacon also polls Docker periodically to
ensure all services are available.

Keys added to etcd have a TTL associated with them. The TTL is used in
conjunction with Beacon's polling to ensure services are cleaned up
automatically should an unexpected failure occur. This could be caused by an
uncontrolled shutdown of the host system, your other datacenter catching fire,
or whatever.

Installing
----------
You can use the go command to get and install the package:

    go get github.com/BlueDragonX/beacon
    go install github.com/BlueDragonX/beacon

Configuring 
-----------
Beacon takes a few commandline options. These are:

`config` - The path to the config file. Defaults to /etc/beacon.yml.
`var` - The container environment variable which holds the hosted services value.
`etcd` - The etcd endpoint. May be provided multiple times.
`docker` - The Docker endpoint.
`prefix` - A prefix to prepend to all etcd key paths.

The config file itself is YAML. It is structured into sections `service`,
`docker`, `etcd`, and `logging`. Go [here][1] for an example.

### service ###
This section defines service configuration including timeouts and service
location. Available parameters are:

- `var` - The name of an environment variable on a container to read service
  configuration from. This defaults to `SERVICES`. The value of this variable
  is a comma separated list of service definitions. The structure of a service
  definition is `name:port/protocol` where `name` is the name of the service,
  `port` is the port the service listens on, and protocol is either `tcp` or
  `udp`. The protocol defaults to `tcp` if omitted. If the container does not
  expose a port for a defined service then it will not be announced.
- `tags` - A list of tags with which to filter the containers to announce
  services for. Containers whose `tags-var` contains one of these tags will
  have its services announced by Beacon.
- `tags-var` - The name of an environment variable on a container to read tags
  from. Tags may be used to filter containers to announce services for. The
  variable contains a comma separated list of tags to flag the container with.
- `hostname` - The hostname the services should be reachable at. This should be
  an address reachable by external clients of your service. Defaults to the
  system hostname.
- `heartbeat` - How often (in seconds) services will be polled. Defaults to 30.

### docker ###
This section configures the connection to Docker. Available parameters are:

- `uri` - The URI to connect to Docker at. Defaults to `unix:///var/run/docker.sock`.

### etcd ###
This section configures the connection to etcd. Available parameters are:

- `uri` - The URI to connect to etcd at. Defaults to `http://172.17.42.1:4001/`.
- `uris` - Connect to multiple etcd nodes. Used as an alternative to `uri` when
  redundancy is called for.
- `prefix` - etcd key paths will be prefixed with this value. Defaults to `beacon`.
- `ttl` - How long (in seconds) a service should remain in etcd after missing a
  heartbeat. Defaults to 30. The etcd TTL is calculated as `heartbeat + ttl`.
- `tls-key` - The path to the TLS private key to use when connecting. Must be
  provided to enable TLS.
- `tls-cert` - The path to the TLS certificate to use when connecting. Must be
  provided to enable TLS.
- `tls-ca-cert` - The path to the TLS CA certificate to use when connecting.
  Must be provided to enable TLS.

### logging ###
This section controls how Beacon outputs logging. Available parameters are:

- `console` - Whether or not to log to the console. Defaults to true.
- `syslog` - Whether or not to log to syslog. Defaults to false.
- `level` - The log level. Valid values are `debug`, `info`, `notice`, `warn`,
  `error`, or `fatal`.

Etcd Keys
---------
Services are announced to etcd under the `etcd.prefix` defined in the config
file. The service paths are structured as follows:

    /<service_name>/<container_id>/<keys>

The following keys are set fore each service/container:

- `container-name` - The hostname of the container the service is running on.
- `container-port` - The port the container exposes for this service.
- `host-name` - The value of `service.hostname`.
- `host-port` - The port on the host that is mapped to this service.
- `protocol` - The protocol of the service. Either `tcp` or `udp`.

There are two index keys which are incremented after services are updated:

- `/_index` - The global index. Incremented after any service is changed.
- `/<service_name>/_index` - Service index. Incremented after that service changes.

License
-------
Copyright (c) 2015 Ryan Bourgeois. Licensed under BSD-Modified. See the
[LICENSE][2] file for a copy of the license.

[1]: https://raw.githubusercontent.com/BlueDragonX/beacon/master/config_example.yml "Beacon Example Config"
[2]: https://raw.githubusercontent.com/BlueDragonX/beacon/master/LICENSE "Beacon License"
