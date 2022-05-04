![TriggerMesh Logo](.assets/triggermesh-logo.png)

[![Release](https://img.shields.io/github/v/release/triggermesh/triggermesh?label=release)](https://github.com/triggermesh/triggermesh/releases)
[![CircleCI](https://circleci.com/gh/triggermesh/triggermesh/tree/main.svg?style=shield)](https://circleci.com/gh/triggermesh/triggermesh/tree/main)
[![Go Report Card](https://goreportcard.com/badge/github.com/triggermesh/triggermesh)](https://goreportcard.com/report/github.com/triggermesh/triggermesh)

The TriggerMesh Cloud Native Integration Platform consists of a set of APIs which allows you to build event-driven
applications. Implemented as a set of Kubernetes CRDs and a Kubernetes controller, it gives you a way to declaratively
define your event sources and event targets, in addition to potential actions needed in your applications: content-based
event filtering, event splitting, event transformation and event processing via functions.

## Getting Started

* [Guides](https://docs.triggermesh.io/guides/creatingasource/)
* [Documentation](https://docs.triggermesh.io)

## Installation

To install TriggerMesh, follow the [installation instructions](https://docs.triggermesh.io/guides/installation/).

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
