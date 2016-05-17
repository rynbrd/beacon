Beacon
======
[![Build Status](https://travis-ci.org/BlueDragonX/beacon.svg?branch=master)](https://travis-ci.org/BlueDragonX/beacon)

Beacon pipes container start/stop events to various systems. Currently it supports Docker as its runtime and Amazon SNS as its backend. More runtimes and backends are currently planned.

How It Works
------------
Beacon listens for events on the configured runtime. When an event is receieved it evaluates a filter expression attached to each of the configured backends. The event is processed by each backend whose filter matches the event.

Beacon allows containers to be grouped into services. The runtime is responsible for identifying the service a container is part of.

Building
--------
Beacon uses Make to drive the test and build process.

To test Beacon:

	make test

to build Beacon:

	make

The build will create a binary called `beacon` in a newly created `bin` directory.

Running
-------
Beacon takes a single command line flag: `-config`. It should be the path to the config file. If not set the default config file is at `/etc/beacon.yml`.

Config File
-----------
The config file is formatted as [YAML][3]. It has sections for the runtime (docker) and backends. An example config file is available [here][2].

Runtimes
--------
Currently Beacon supports a single runtime: Docker.

The Docker runtime is configured with a socket, host IP, and label. The socket is of type `unix://` or `tcp://` and is used to connect to the Docker daemon. Port bindings which listen on 0.0.0.0 are assigned the host IP. Lastly the label is the name of the lable containing the name of the service. Events are ignored for containers which do not have this label.

A config file snippet for Docker:

	docker:
	  socket: unix:///var/run/docker.sock
	  hostip: 169.254.12.152
	  label: service

The Docker runtime can be configured to send stop events for all running containers when Beacon stops. This is done by setting the `stop-on-exit` value to true:

	docker:
	  socket: unix:///var/run/docker.sock
	  hostip: 169.254.12.152
	  label: service
	  stop-on-exit: true

Backends
--------
Currently Beacon supports two backends: `sns` and `debug`.

### SNS
The `sns` backend queues events to an AWS SNS topic. The SNS backend is configured with a region and topic ARN.

A config file snippet for SNS:

	backends:
	- sns:
		region: us-east-1
		topic: arn:aws:sns:us-east-1:698519295917:TestTopic
	  filter:
		group: ops

The SNS message is a JSON encoded event. An example follows:

	{
		"Action": "start",
		"Container": {
			"ID": "512b64138152",
			"Service": "www",
			"Labels": {
				"service": "www",
				"service_port": "80",
			},
			"Bindings": [
				{
					"HostIP": "169.254.12.152",
					"HostPort": 54698,
					"ContainerPort": 80,
					"Protocol": "tcp"
				}
			]
		}
	}

### Debug
The `debug` backend prints events to the log.

A config file snippet for the debug:

	backends:
	- debug: {}

License
-------
Copyright (c) 2015 Ryan Bourgeois. Licensed under BSD-Modified. See the [LICENSE][1] file for a copy of the license.

[1]: https://raw.githubusercontent.com/BlueDragonX/beacon/master/LICENSE "License"
[2]: https://raw.githubusercontent.com/BlueDragonX/beacon/master/config.yml "Example Config File"
[3]: http://yaml.org/ "YAML"
