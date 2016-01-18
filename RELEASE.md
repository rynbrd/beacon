## v2.1.4
- Remove -a flag for build.

## v2.1.3
+ Add Makefile.

## v2.1.2
* Update go-log and go-settings deps.
* Use latest dockerclient. 

## v2.1.1
* Include TTL when setting keys.

## v2.1.0
+ Add support for JSON etcd values.
+ Add option '-etcd-format'.
* Rename option '-prefix' to '-etcd-prefix'.
* Replace 'etcd.protocol' with 'etcd.format' config option.
* Default etcd key format to JSON.

## v2.0.1
* Unpin go-etcd library.

## v2.0.0
* Improved architecture to support future container runtimes and discovery backends.
* Simplify etcd directory layout.
* New configuration file format.
* Do not require a configuration file.
* Use go-log and go-settings for logging and configuration.
+ Support command line configuration.
+ Test coverage over core components.

## v1.1.7
* Use pinned go-etcd version.
* Fix a bug where a Docker events prevent polling to occur.

## v1.1.6
* Auto-remove trailing slash from etcd URIs.

## v1.1.5
* Use rolled-back dockerclient.
* Support older Docker versions (API v1.10).

## v1.1.4
* Use local fork of dockerclient to avoid even further breakage.

## v1.1.3
* Fix further bugs associated with DockerClient API breakage.

## v1.1.2
* Fix bug caused by DockerClient API breakage.
