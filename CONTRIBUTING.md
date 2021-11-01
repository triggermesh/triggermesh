# Contributing to TriggerMesh

Welcome, contributors! :wave:

Thank you for taking the time to go through this document, which suggests a few guidelines for contributing to the
TriggerMesh open source platform.

We define _contributions_ as:

- Bug reports
- Feature and enhancement requests
- Code submissions
- Any participation in discussions within the TriggerMesh community

## Contents

1. [Code of Conduct](#code-of-conduct)

1. [Submitting Contributions](#submitting-contributions)
   - [Reporting Bugs](#reporting-bugs)
   - [Requesting Features and Enhancements](#requesting-features-and-enhancements)
   - [Submitting Code Changes](#submitting-code-changes)

1. [Development Guidelines](#development-guidelines)

   1. [Prerequisites](#prerequisites)
      - [Go Toolchain](#go-toolchain)
      - [Kubernetes](#kubernetes)
      - [Knative Serving](#knative-serving)
      - [ko](#ko)

   1. [Running the Controller](#running-the-controller)
      - [Controller Configuration](#controller-configuration)
      - [Locally](#locally)
      - [Inside a Cluster](#inside-a-cluster)

   1. [Adding a Component](#adding-a-component)
      - [API Documentation](#api-documentation)

## Code of Conduct

Although this project is not part of the [CNCF][cncf], we abide by its [Code of Conduct][cncf-coc], and expect all
contributors to uphold this code. Please report unacceptable behavior to <info@triggermesh.com>.

## Submitting Contributions

The guidelines below aim at ensuring that maintainers can understand submissions as quickly and effortlessly as
possible, whether these are questions, issue reports, feature requests, or code contributions. The golden rule is: the
clearer the information, the faster the resolution :rocket:.

### Reporting Bugs

Bugs, or any kind of issue you encounter with the TriggerMesh platform, can be reported using [GitHub Issues][gh-issue].

Before opening a new issue, kindly [search for a few keywords][gh-search] related to the problem you encountered, just
to ensure a similar report hasn't already been submitted. Didn't find anything relevant? Great! :+1: Let's create that
issue.

:information_source: _If you suspect or discover a security vulnerability in the TriggerMesh software, please do not
disclose it publicly via a GitHub issue. Instead, please report it to <info@triggermesh.com> so that maintainers can
address it within the shortest possible delay._ :lock::hourglass:

#### Issue Description

A good bug report starts with a clear and descriptive title. Avoid overly generic titles such as "bug in component X",
or raw outputs from error logs. Indicate _what_ is failing and, if known, under _what circumstances_ it is failing (e.g.
"Component X panics when environment variable Y is not set").

Although there is no enforced template for submitting issues, we do recommend including the following information:

- A detailed description, in plain English, of the behaviour you are observing, and what you expected instead.
- The release version of the TriggerMesh platform the problem can be observed with (or software revision if the software
  was built from source).
- The component that is affected (a specific event source/target/router, a function runtime, etc.)
- A configuration snippet that can be used to reproduce the issue.
- If some preliminary setup of a third-party service was performed, please describe those steps.
- The error messages you are seeing, if any.
  - :bulb: Most errors are reported via Kubernetes API events and object statuses. Both can be obtained using the
    [kubectl describe][k-describe] command (e.g. `kubectl describe zendesksources/my-helpdesk`).

Remember, anything that allows maintainers to reproduce the problem from the _initial_ issue description is another day
saved going back and forth to the issue's comments to ask for additional information, leading to _your_ issue being
solved faster! :raised_hands:

#### Code Styling

To ensure the indentation of command outputs and the highlighting of code snippets are preserved inside the text of
GitHub issues, we recommend wrapping them inside [Fenced Code Blocks][gh-fenced] using the triple backticks notation
(` ``` `).

Examples:

<table>

<thead>
<tr>
<th>Raw Markdown</th>
<th>Rendered code block</th>
</tr>
</thead>

<tbody>
<tr>
<td>

````
```
[2021/12/01 15:43:04] Some log output
```
````

</td>
<td>

```
[2021/12/01 15:43:04] Some log output
```

</td>
</tr>

<tr>
<!–– NOTE(antoineco): empty row to prevent alternate row highlight a.k.a. "zebra striping" -->
</tr>

<tr>
<td>

````
```yaml
# My Kubernetes manifest
apiVersion: triggermesh.io/v1
```
````

</td>
<td>

```yaml
# My Kubernetes manifest
apiVersion: triggermesh.io/v1
```

</td>
</tr>
</tbody>

</table>

### Requesting Features and Enhancements

Features and enhancements can be reported using [GitHub Issues][gh-issue].

Similarly to [bug reports](#reporting-bugs), we kindly ask you to [search for a few keywords][gh-search] related to your
suggestion before opening a new issue, just to ensure a similar request hasn't already been submitted. In case that
search doesn't yield any relevant result, let's go ahead and create that issue. :memo:

Provide a detailed description, in plain English, of the result you are expecting by submitting your request:

- If you are asking for an enhancement to an existing component, be specific about which component. If possible, include
  links to any external resources that may help maintainers get a clearer understanding of the desired outcome.
- If you are asking for a new feature, please clarify the nature of that feature. Examples include:
  - A new integration with a third-party service.
  - A new data processor.
  - A new authentication method.
  - ...

The clearer the request, the easier it is for maintainer to discuss a potential design and implementation!
:raised_hands:

### Submitting Code Changes

Code submissions can be proposed using [GitHub Pull Requests][gh-pr].

Whenever a pull request is opened, and every time it is updated by pushing new commits, the CI pipeline performs some
static code analysis on the submitted code revision to ensure that certain [code styles][ci-linters] are respected. All
status checks must be passing for a pull request to be considered by maintainers. :heavy_check_mark:

Small, non-breaking changes, can be submitted spontaneously without prior discussion with maintainers, providing that
they include a clear justification of their potential relevance to the project.

Larger changes such as new features, or changes which impact the current behaviour of certain TriggerMesh components,
should be socialized and discussed with maintainers in a [GitHub Issue][gh-issue] or via Slack
[@triggermesh-community][tm-slack]. Nobody likes seeing a submission getting rejected because it was not aligned with
the project's goals or standards! :disappointed:

If you read that far and are feeling ready to submit your first code contribution, congratulations! :heart: Read on, the
following section about [Development Guidelines](#development-guidelines) explains what our standard development
environment looks like.

## Development Guidelines

### Prerequisites

- Go toolchain
- Kubernetes
- Knative Serving
- ko

#### Go Toolchain

TriggerMesh is written in [Go][go].

The Go toolchain is required in order to be able to compile the TriggerMesh code and run its automated tests.

The currently recommended version can be found inside the [`go.mod`](go.mod) file, based on the `go` directive.

#### Kubernetes

TriggerMesh runs on top of the [Kubernetes][k8s] platform.

Any certified Kubernetes distribution can run TriggerMesh, whether it is running locally (e.g. using [`kind`][k8s-kind],
inside a virtual machine, ...) or remotely (e.g. inside a cloud provider, in your own datacenter, ...).

The currently recommended version can be found inside the [`go.mod`](go.mod) file, based on the version of the
`k8s.io/api` module dependency.

#### Knative Serving

Most components of the TriggerMesh platform run as [Knative Services][kn-serving].

Knative can be installed via different methods, which are described inside the [Knative Administration Guide][kn-admin].
It must be deployed inside the same Kubernetes cluster that runs TriggerMesh.

The currently recommended version can be found inside the [`go.mod`](go.mod) file, based on the version of the
`knative.dev/serving` module dependency.

:information_source: _The [Getting Started with Knative][kn-quickstart] guide contains instructions for deploying a
Docker-based local environment, bootstrapped using [`kind`][k8s-kind], that includes both Kubernetes and Knative. We
endorse this approach as an alternative to a manual Knative installation for local development._

#### ko

[`ko`][ko] is a tool which allows developers to package Go projects as container images and deploy them to Kubernetes in
a single command, all of this without requiring Docker to be installed.

TriggerMesh relies on `ko` extensively, both for development purposes and for its own releases.

We recommend using version `v0.9.0` or greater.

### Running the Controller

#### Controller Configuration

The TriggerMesh controller reads its configuration from:

- The environment, for immutable settings.
- [ConfigMap][k8s-cmap] objects deployed to Kubernetes, for dynamic settings.

##### Configuration Read From the Environment

The environment variables below must be [exported][sh-export] to the environment the controller process runs in.
Typically, this is done either from the shell, or inside a Kubernetes [Pod manifest][k8s-env] when running inside a
container.

- `SYSTEM_NAMESPACE`

   The Kubernetes namespace in which the controller reads its configuration for logging and observability from
   (ConfigMap objects). This can potentially be set to any existing namespace, including `default`, since the controller
   falls back to a default configuration if the expected ConfigMaps are missing. _More details about ConfigMaps in the
   next section._

- `METRICS_DOMAIN`

   _Only required when observability is **enabled**. Please refer to the next section about ConfigMaps._
   The domain to use for surfacing [Prometheus metrics][prom]. In a development environment, the actual value of this
   variable does not matter and a placeholder can be used (e.g. `triggermesh.local`), unless you are actively collecting
   those metrics for your own analysis purposes and need to identify the ones originating from TriggerMesh.

Additionally, the desired build version (container image) of each TriggerMesh components can be overridden by
environment variables. The full list can be found inside the [controller's Deployment manifest][tm-allenvs]. By default,
the controller uses the latest tagged public image from `gcr.io/triggermesh` for any given component.

To use your own image instead (e.g. the result of a development build), simply set the corresponding environment
variable to the reference of the image to use. For example:

```sh
# Use a custom build of the "Zendesk" event source adapter.
export ZENDESKSOURCE_IMAGE=docker.io/myuser/zendesksource:devel
```

##### Configuration Read From ConfigMaps

Some aspects of TriggerMesh can be configured via ConfigMap objects:

- Logging (log level, log format, ...)
- Observability (metrics backend, Go runtime tracing, ...)
- Distributed tracing (tracing backend, sampling, ...)

The ConfigMaps for configuring these aspects are respectively named `config-logging`, `config-observability` and
`config-tracing`. They are read from the Kubernetes namespace identified by the `SYSTEM_NAMESPACE` environment variable
described in the previous section.

If present, the settings contained in these ConfigMaps are applied dynamically to TriggerMesh components at runtime,
whenever they are changed by the user. Otherwise, TriggerMesh falls back to [sane logging defaults][zapdriver], and does
not enable any observability or distributed tracing backends.

For a description of the configuration settings which are currently supported, please refer to the `logging.yaml`,
`observability.yaml` and `tracing.yaml` manifest files inside the [Knative Eventing source repository][kn-cmaps].

#### Locally

It is possible to run the TriggerMesh controller (`./cmd/triggermesh-controller`) locally, and let it operate against
_any_ Kubernetes cluster, whether this cluster is running locally (e.g. inside a virtual machine) or remotely (e.g.
inside a cloud provider).

:warning: **Before running the controller in your local environment, make sure no other instance of the TriggerMesh
controller is currently running inside the target cluster! This could prevent your local instance from performing any
work due to the leader-election mechanism, or worse, result in multiple controllers performing conflicting changes
simultaneously to the same objects.**

Providing that _(1)_ the local environment is configured with a valid [kubeconfig][k8s-kubecfg] and _(2)_ the
aforementioned [mandatory environment variables](#configuration-read-from-the-environment) are exported, running the
controller locally from the current development branch is as simple as executing:

```console
$ go run ./cmd/triggermesh-controller
2021/10/12 13:35:55 Registering 4 clients
2021/10/12 13:35:55 Registering 4 informer factories
2021/10/12 13:35:55 Registering 68 informers
2021/10/12 13:35:55 Registering 63 controllers
{"severity":"INFO","timestamp":"2021-10-12T13:35:55","caller":"logging/config.go:116","message":"Successfully created the logger."}
{"severity":"INFO","timestamp":"2021-10-12T13:35:55","caller":"logging/config.go:117","message":"Logging level set to: info"}
...
```

#### Inside a Cluster

One can deploy/update all Kubernetes objects to a running cluster and build/push the associated container images in a
single command using `ko`:

```console
$ ko apply --local -f config/
2021/10/12 13:10:35 Using base gcr.io/distroless/static:nonroot for github.com/triggermesh/triggermesh/cmd/triggermesh-controller
2021/10/12 13:10:39 Building github.com/triggermesh/triggermesh/cmd/triggermesh-controller for linux/amd64
2021/10/12 13:11:54 Loading ko.local/triggermesh-controller-891f0506b996f2dab5fd9aae5acdf6bd:2065[...]22c
2021/10/12 13:11:57 Loaded ko.local/triggermesh-controller-891f0506b996f2dab5fd9aae5acdf6bd:2065[...]22c
2021/10/12 13:11:57 Adding tag latest
2021/10/12 13:11:57 Added tag latest
...
deployment.apps/triggermesh-controller created
```

The `triggermesh-controller` Deployment, along with the other Kubernetes objects which constitute the TriggerMesh
platform, are deployed to the `triggermesh` namespace:

```console
$ kubectl -n triggermesh get deployment/triggermesh-controller
NAME                     READY   UP-TO-DATE   AVAILABLE   AGE
triggermesh-controller   1/1     1            1           1m
```

:information_source: _Although `ko` does not make use of Docker to build and push container images, the `--local` flag
has the particularity that built images are loaded directly into the container runtime (e.g. Docker), instead of being
pushed to a container registry. Therefore, the container runtime used by the local Kubernetes installation must be
accessible from the developer's shell. On `minikube`, for example, it is assumed that the environment variables defined
by `minikube docker-env` are exported. Please refer to the `ko` documentation for further usage instructions._

### Adding a Component

For a comprehensive and step-by-step tutorial about writing an event source for Knative, please refer to [Creating an
event source by using the sample event source][kndoc-source] in the Knative documentation. The concepts explained in
this tutorial apply to the vast majority of TriggerMesh integrations.

The conventions which apply specifically to TriggerMesh are detailed in the following sections.

#### API Documentation

TriggerMesh's reference API documentation is generated based on the [Go documentation][go-doc] of the types found inside
`*_types.go` files, in sub-packages of [`pkg/apis/`](pkg/apis/).

Provide as many details as possible and pay extra attention to the format of these code comments while adding a new type
or editing an existing one, since they are directly translated to user-facing documentation. Everybody likes reading
clear API docs! :sparkles:


[cncf]: https://www.cncf.io/
[cncf-coc]: https://github.com/cncf/foundation/blob/master/code-of-conduct.md

[tm-slack]: https://triggermesh-community.slack.com
[tm-apidocs]: https://docs.triggermesh.io/apis/apis/

[gh-issue]: https://github.com/triggermesh/triggermesh/issues
[gh-search]: https://github.com/triggermesh/triggermesh/issues?q=
[gh-fenced]: https://docs.github.com/en/github/writing-on-github/working-with-advanced-formatting/creating-and-highlighting-code-blocks
[gh-pr]: https://github.com/triggermesh/triggermesh/pulls

[go]: https://golang.org/
[go-mod]: https://github.com/triggermesh/triggermesh/blob/main/go.mod#L3
[go-doc]: https://go.dev/blog/godoc

[ko]: https://github.com/google/ko
[prom]: https://prometheus.io/docs/introduction/overview/
[zapdriver]: https://github.com/blendle/zapdriver#readme

[k8s]: https://kubernetes.io/
[k8s-cmap]: https://kubernetes.io/docs/concepts/configuration/configmap/
[k8s-env]: https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/
[k8s-kubecfg]: https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/
[k8s-kind]: https://kind.sigs.k8s.io/
[k-describe]: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#describe

[kn-quickstart]: https://knative.dev/docs/getting-started/
[kn-admin]: https://knative.dev/docs/admin/
[kn-serving]: https://knative.dev/docs/serving/
[kndoc-source]: https://knative.dev/docs/developer/eventing/sources/creating-event-sources/writing-event-source/
[kn-cmaps]: https://github.com/knative/eventing/tree/v0.26.0/config/core/configmaps

[sh-export]: https://www.gnu.org/software/bash/manual/html_node/Environment.html
[tm-allenvs]: https://github.com/triggermesh/triggermesh/blob/main/config/500-controller.yaml#L59-L999
[ci-linters]: https://golangci-lint.run/usage/linters/#enabled-by-default-linters
