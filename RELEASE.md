## v1.1.7
* Use time.Tick instead of time.After to poll. This fixes a bug where a flood
  of docker events would cause poll to never execute which would in turn cause
  the etcd TTLs to expire.

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
