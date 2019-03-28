---
title: "Dependencies"
linkTitle: "Dependencies"
weight: 4
description: >
  A short lead descripton about this section page. Text here can also be **bold** or _italic_ and can even be split over multiple paragraphs.
---

## Open Source Software integrations

### Structured logging

Logging for Open Match uses the [Golang logrus module](https://github.com/sirupsen/logrus) to provide structured logs. Logs are output to `stdout` in each component, as expected by Docker and Kubernetes. Level and format are configurable via config/matchmaker_config.json. If you have a specific log aggregator as your final destination, we recommend you have a look at the logrus documentation as there is probably a log formatter that plays nicely with your stack.

### Instrumentation for metrics

Open Match uses [OpenCensus](https://opencensus.io/) for metrics instrumentation. The [gRPC](https://grpc.io/) integrations are built-in, and Golang redigo module integrations are incoming, but [haven't been merged into the official repo](https://github.com/opencensus-integrations/redigo/pull/1). All of the core components expose HTTP `/metrics` endpoints on the port defined in `config/matchmaker_config.json` (default: 9555) for Prometheus to scrape. If you would like to export to a different metrics aggregation platform, we suggest you have a look at the OpenCensus documentation &mdash; there may be one written for you already, and switching to it may be as simple as changing a few lines of code.

**Note:** A standard for instrumentation of MMFs is planned.

### Redis setup

By default, Open Match expects you to run Redis *somewhere*. Connection information can be put in the config file (`matchmaker_config.json`) for any Redis instance reachable from the [Kubernetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/). By default, Open Match sensibly runs in the Kubernetes `default` namespace. In most instances, we expect users will run a copy of Redis in a pod in Kubernetes, with a service pointing to it.

* HA configurations for Redis aren't implemented by the provided Kubernetes resource definition files, but Open Match expects the Redis service to be named `redis`, which provides an easier path to multi-instance deployments.
* 