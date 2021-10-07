# Tekton Pipeline Event Target for Knative Eventing

This event target integrates with Tekton pipelines using received CloudEvent
messages to trigger a pipeline or task run.

## Prerequisites

Tekton Pipelines must be installed on the same cluster as the Tekton Pipeline
event target.  For instructions on how to install Tekton, please see their
[installation guide](https://tekton.dev/docs/getting-started/).

In addition, the event target assumes the target pipelines and tasks
exist in the cluster.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make tekton-target-adapater && ./_output/tekton-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`    - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`  - Define the default logging configuration
  - `K_METRICS_CONFIG='''` - Define the prometheus metrics configuration

## Creating a Tekton Pipeline Target

Once the Tekton Pipeline Target has been deployed along with the
requisite Tekton objects, then the target can be created by defining a TektonTarget object:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: TektonTarget
metadata:
  name: <TARGET-NAME>
```

## Tekton Target as an Event Sink

Tekton Target is addressable and can be used as an event sink for other
Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: <TRIGGER-NAME>
spec:
  broker: <BROKER-NAME>
  filter:
    attributes:
      type:  <MESSAGE-TYPES-TEKTON-FORMATTED>
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: TektonTarget
      name: <TARGET-NAME>
```

## Triggering a Tekton Pipeline Run via the Target

The Tekton Pipeline Target can be triggered as a normal web service using a
tool such as `curl` within the cluster.  A sample message would resemble the
following:

```console
curl -v http://tektontarget-helloworld5d0adf0209a48c23fa958aa1b8ecf0b.default.svc.cluster.local \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: io.triggermesh.tekton.run" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"buildtype": "task","name": "tekton-test","params":{"greeting":"Hi from TriggerMesh"}}'
```

## Tekton Pipeline Target Message Format

The Tekton Pipeline Target message format must contain at least:
  - buildtype (can be one of task or pipeline)
  - name (this is the name of the pipeline or task object being invoked)
  - params (optional JSON object of key/value pairs that the Tekton CRD is expecting)

## Reaping prior Tekton TaskRuns and PipelineRuns

To allow for reaping of old run objects, the `TektonTarget` Spec supports defining
a duration interval (in the form of `\d+[mhd]` for minute, hour, or day) for how
long to keep the run objects before purging.

  - `reapPolicy.success` Age of run objects to keep that succeeded
  - `reapPolicy.fail` Age of the run objects to keep that failed

To trigger the reaping, a [CloudEvent][ce] type of `io.triggermesh.tekton.reap`
must be sent to the target.

[ce]: https://cloudevents.io/
