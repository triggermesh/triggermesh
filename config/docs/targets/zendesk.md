# Zendesk Event Target for Knative Eventing

This event target integrates with Zendesk, using received CloudEvent messages to
create a Zendesk ticket or add a tag to a pre-existing ticket. 

## Contents

- [Zendesk Event Target for Knative Eventing](#zendesk-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Creating a Zendesk Target](#creating-a-zendesk-target)
    - [Status](#status)
    - [Zendesk Target as an Event Sink](#zendesk-target-as-an-event-sink)
    - [Sending Messages to the Zendesk Target](#sending-messages-to-the-zendesk-target)

## Prerequisites

A Zendesk API key is required to utilize this target. The steps to obtain a key
are outlined in the [Zendesk API Document](https://support.zendesk.com/hc/en-us/articles/226022787-Generating-a-new-API-token-).

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make zendesk-target-adapater && ./_output/zendesk-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`    - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`  - Define the default logging configuration
  - `K_METRICS_CONFIG='''` - Define the prometheus metrics configuration
  - `TOKEN`                - Zendesk API Token
  - `EMAIL`                - Default email address to use as the ticket submitter
  - `SUBDOMAIN`            - Zendesk subdomain for the owning account
  - `SUBJECT`              - (optional) Subject describing the ticket

A full deployment example is located in the [samples](../samples/zendesk) directory

### Status

A ZendeskTarget requires a secret : `token`, and subdomain `subdomain`. Once they
are present, the Target will create a Knative Service. The global controller
logs and events can provide detailed information about the process. A Status
summary in included with the ZendeskTarget object to provide details on the 
conditions the target needs.

When ready the `status.address.url` will provide the internal point where the CloudEvents should target.

## Creating a Zendesk Target

Once the Zendesk Target Controller has been deployed, A Kubernetes secret named `zendesk` must be created.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: zendesk
type: Opaque
stringData:
   token: <Zendesk token>
```

A ZendeskTarget can be created by using the following YAML:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: ZendeskTarget
metadata:
  name: triggermesh-zendesk
spec:
  #subject provides a default Subject for new Zendesk tickets. Optional
  subject: '' #Example: tmTickets0
  subdomain: '' #Example: tmdev1
  email: '' #Example: jeff@triggermesh.com
  token:
     secretKeyRef:
       name: zendesktargetsecret
       key: token
```

`subdomain,` `email,` &  `token` are ***ALL REQUIRED*** to deploy.

### Zendesk Target as an Event Sink

A Zendesk Target is addressable which means it can be used as a Sink for Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: zendesk-sample-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: ZendeskTarget
      name: triggermesh-zendesk
```

A sample sink binding to a Zendesk Target deployment. 

```yaml
apiVersion: sources.triggermesh.io/v1alpha1
kind: <Sample Source>
metadata:
  name: <Sample Source Name>
spec:
  sampleToken:
    secretKeyRef:
      name: <sample>
      key: <sample key>
  sink:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: ZendeskTarget
      name: triggermesh-zendesk
```

### Sending Messages to the Zendesk Target

A Zendesk Target will ONLY accept
[CloudEvents](https://github.com/cloudevents/spec) with a "Ce-Type" of either
`com.zendesk.ticket.create` OR `com.zendesk.tag.create`

* Event's of type `com.zendesk.ticket.create` Expect both a `subject` and `body` to be preset.

  - **Example of type : `com.zendesk.ticket.create`**
    ```sh
    curl -v https://zendesktarget-triggermesh-zendesk.jnlasersolutions.dev.munu.io  \
    -H "Content-Type: application/json" \
    -H "Ce-Specversion: 1.0" \
    -H "Ce-Type: com.zendesk.ticket.create" \
    -H "Ce-Source: some.origin/intance" \
    -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
    -d '{"subject": "Hello", "body" : "World"}'
    ```

* Event's of type `com.zendesk.tag.create` Expect both a `id` and `tag` to be preset.
  - **Example of type : `com.zendesk.tag.create`**
    ```sh
    curl -v https://zendesktarget-triggermesh-zendesk.jnlasersolutions.dev.munu.io  \
    -H "Content-Type: application/json" \
    -H "Ce-Specversion: 1.0" \
    -H "Ce-Type: com.zendesk.tag.create" \
    -H "Ce-Source: some.origin/intance" \
    -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
    -d '{"id":81 , "tag":"triggermesh"}'
    ```



