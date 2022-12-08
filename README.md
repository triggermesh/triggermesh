![TriggerMesh Logo](.assets/triggermesh-logo.png)

[![Release](https://img.shields.io/github/v/release/triggermesh/triggermesh?label=release)](https://github.com/triggermesh/triggermesh/releases)
[![CircleCI](https://circleci.com/gh/triggermesh/triggermesh/tree/main.svg?style=shield)](https://circleci.com/gh/triggermesh/triggermesh/tree/main)
[![Go Report Card](https://goreportcard.com/badge/github.com/triggermesh/triggermesh)](https://goreportcard.com/report/github.com/triggermesh/triggermesh)
[![Slack](https://img.shields.io/badge/Slack-Join%20chat-4a154b?style=flat&logo=slack)](https://join.slack.com/t/triggermesh-community/shared_invite/zt-1kngevosm-MY7kqn9h6bT08hWh8PeltA)

TriggerMesh is an open-source alternative to AWS EventBridge. It lets you easily capture events from many sources, uniformily transform and route them, and reliably deliver them to event consumers. 

TriggerMesh comes in two main flavours. This repository provides TriggerMesh on Kubernetes, implemented as a set of Kubernetes CRDs and a Kubernetes controller. It gives you a way to declaratively define your event sources and event targets, transformation and routing.

TriggerMesh also provides a command line interface called [tmctl](https://github.com/triggermesh/tmctl) which makes it easy to run TriggerMesh locally, on any laptop with just Docker as a prerequisite. 

## Getting Started

* For first-time users, we recommend the [Quickstart with tmctl](https://docs.triggermesh.io/get-started/quickstart/)
* [Documentation](https://docs.triggermesh.io)

## Installation

To install TriggerMesh on Kubernetes, follow the [installation instructions](https://docs.triggermesh.io/installation/).

### TL;DR

Register TriggerMesh APIs by deploying the Custom Resources Definitions:

```shell
kubectl apply -f https://github.com/triggermesh/triggermesh/releases/latest/download/triggermesh-crds.yaml
```

Deploy the platform:

```shell
kubectl apply -f https://github.com/triggermesh/triggermesh/releases/latest/download/triggermesh.yaml
```

## Contributing

Please refer to our [guidelines for contributors](CONTRIBUTING.md).

## Commercial Support

TriggerMesh Inc. offers commercial support for the TriggerMesh platform. Email us at <info@triggermesh.com> to get more
details.

## License

This software is licensed under the [Apache License, Version 2.0][asl2].

Additionally, the End User License Agreement included in the [`EULA.pdf`](EULA.pdf) file applies to compiled
executables and container images released by TriggerMesh Inc.

[asl2]: https://www.apache.org/licenses/LICENSE-2.0
