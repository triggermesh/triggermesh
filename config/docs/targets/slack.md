# Slack Event Target for Knative Eventing

This event target uses CloudEvents to consume the Slack WebAPI.
Supported methods are:

- chat.postMessage
- chat.scheduleMessage
- chat.update

## Contents

- [Slack Event Target for Knative Eventing](#slack-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Create Slack Target Integration](#create-slack-target-integration)
    - [Creating the Slack App Bot and Token Secret](#creating-the-slack-app-bot-and-token-secret)
    - [Creating a Slack Target](#creating-a-slack-target)
  - [Using Web API Methods from the Slack Target](#using-web-api-methods-from-the-slack-target)
    - [Send message](#send-message)
    - [Send Scheduled Message](#send-scheduled-message)
    - [Update Message](#update-message)

## Prerequisites

A Slack API token is required to utilize this target. For more information on
how to obtain one, see the [Slack Developer's Guide](https://api.slack.com/start)

The application created in Slack for this integration will need to be added
to the scopes that satisfy the web API methods used. See [Sending Slack Operations](#sending-a-slack-message-via-the-target)
to determine which scopes will be required per operation.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make slack-target-adapater && ./_output/slack-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`    - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`  - Define the default logging configuration
  - `K_METRICS_CONFIG='''` - Define the prometheus metrics configuration
  - `SLACK_TOKEN`          - Slack API token associated with the application provisioned in the prerequisites

A full deployment example is located in the [samples](../samples/slack) directory

## Create Slack Target Integration

Deploy the Slack source in 2 steps:

1. Create the Slack App Bot and the corresponding secret.
1. Create the Slack Target.

### Creating the Slack App Bot and Token Secret

1. Create a new Slack App at https://api.slack.com/apps
1. From Basic Information, select the Permissions pane.
1. At Bot Token scopes add those that your bot might need. Be aware that we support a subset of the method at those scopes.
1. Select Install App, then install it at your workspace.
1. Copy the Bot OAuth Access token, it should begin with `xoxb-...`
1. Create the secret to be used by the Slack Target:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack
type: Opaque
stringData:
  token: <REPLACE ME WITH A REAL SLACK TOKEN>
```

### Creating a Slack Target

The Slack Target is a service which is able to receive CloudEvents and
transform them into method calls at the Slack API. It needs the aforementioned
secret to be created:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: SlackTarget
metadata:
  name: triggermesh-slack
spec:
  token:
    secretKeyRef:
      name: slack
      key: token
```

Once created the Slack Target service will expose an endpoint to receive
CloudEvents, you can manually send requests to that endpoint or use it as the
Sink for objects like a Broker.

## Using Web API Methods from the Slack Target

CloudEvents consumed by this target should include a valid JSON message
containing the required fields for the Slack Method. Refer to the Slack
documentation link included in each section for information on the expected fields.

### Send Message

- Slack docs https://api.slack.com/methods/chat.postMessage
- Needs chat:write

```sh
curl -v http://localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: com.slack.webapi.chat.postMessage" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: aabbccdd11223344" \
 -d '{"channel":"C01112A09FT", "text": "Hello from TriggerMesh!"}'
```

### Send Scheduled Message

- Slack docs https://api.slack.com/methods/chat.scheduleMessage
- Use with a scheduled future epoch.
- Needs chat:write

```sh
curl -v http://localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: com.slack.webapi.chat.scheduleMessage" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: aabbccdd11223344" \
 -d '{"channel":"C01112A09FT", "text": "Hello from scheduled TriggerMesh!", "post_at": 1593430770}'
```

### Update Message

- Slack docs https://api.slack.com/methods/chat.update
- Use with an existing message timestamp.
- Needs chat:write

```sh
curl -v http://localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: com.slack.webapi.chat.update" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: aabbccdd11223344" \
 -d '{"channel":"C01112A09FT", "text": "Hello from updated2 TriggerMesh!", "ts":"1593430770.001300"}'
```
