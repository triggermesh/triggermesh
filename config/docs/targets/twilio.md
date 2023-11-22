# Twilio Event Target for Knative Eventing

This event target integrates with Twilio using received CloudEvents to send SMS messages.

## Contents

- [Twilio Event Target for Knative Eventing](#twilio-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Creating a Twilio Target](#creating-a-twilio-target)
    - [Status](#status)
    - [Twilio Target as an Event Sink](#twilio-target-as-an-event-sink)
    - [Sending SMS to a Twilio Target](#sending-sms-to-a-twilio-target)

## Prerequisites

A Twilio account is required to run this target:

* Register a Twilio account
* Purchase a phone number with
* Retrieve from Twilio Dashbard Account SID
* Retrieve from Twilio Dashbard Auth Token

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make twilio-target-adapater && ./_output/twilio-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`    - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`  - Define the default logging configuration
  - `K_METRICS_CONFIG='''` - Define the prometheus metrics configuration
  - `TWILIO_SID`           - Twilio String Identifier
  - `TWILIO_TOKEN`         - Twilio API token
  - `TWILIO_DEFAULT_FROM`  - Default number to originate the SMS from
  - `TWILIO_DEFAULT_TO`    - Default number to send the SMS to 

A full deployment example is located in the [samples](../samples/twilio) directory

## Creating a Twilio Target

Integrations can be created by adding TwilioTargets objects.

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: TwilioTarget
metadata:
  name: <TARGET-NAME>
spec:
  defaultPhoneFrom: "<PHONE-FROM>"
  defaultPhoneTo: "<PHONE-TO>"
  sid:
    secretKeyRef:
      name: "<YOUR-SID-SECRET>"
      key: "<YOUR-SID-SECRET-KEY>"
  token:
    secretKeyRef:
      name: "<YOUR-TOKEN-SECRET>"
      key: "<YOUR-TOKEN-SECRET-KEY>"
```

Although `defaultPhoneFrom` is not mandatory it will usually be configured by
matching the phone number purchased with Twilio.

`defaultPhoneTo` will normally not be informed unless the desire is to send
all messages to the same phone number by default.

Both configurations can be overridden by every CloudEvent message received by the Target.

Refer to [Twilio docs for number formating](https://www.twilio.com/docs/lookup/tutorials/validation-and-formatting?code-sample=code-lookup-with-international-formatted-number).

### Status

TwilioTarget will require two Secrets to be provided: SID and Token.  Once
they are present, a Knative service will be created. The global controller
logs and events can provide detailed information about the process. A Status
summary added to the TwilioTarget object provides the conditions the
target needs.

When the target is ready, the `status.address.url` will point to the
internal location where the CloudEvents will be sent.

### Twilio Target as an Event Sink

Twilio Target is addressable, which means it can be used as a Sink for Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: <TRIGGER-NAME>
spec:
  broker: <BROKER-NAME>
  filter:
    attributes:
      type: <MESSAGE-TYPES-TWILIO-FORMATTED>
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: TwilioTarget
      name: <TARGET-NAME>
```

### Sending SMS to a Twilio Target with event type `io.triggermesh.twilio.sms.send`

Twilio Target expect a JSON payload from the CloudEvent that includes:

* `message`: text to be sent.
* `media_urls`: array of URLs pointing to JPG, GIF or PNG resources.
* `from`: phone sourcing the communication. Optional if provided by the TWilioTarget.
* `to`: phone destination. Optional if provided by the TwilioTarget.

You can use `curl` from a container in the cluster pointing to the TwilioTarget exposed URL:

```console
curl -v http://twiliotarget-int1-8dc3abc7d44bdd0130bd0a311bea272f.knative-samples.svc.cluster.local
 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: io.triggermesh.twilio.sms.send" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"message":"Hello from TriggerMesh using Twilio!","to": "+1111111111"}'
```
