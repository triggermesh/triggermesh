# SendGrid Event Target for Knative Eventing

This event target integrates with SendGrid by using received CloudEvent messages to send an email.

## Prerequisites

A SendGrid account and API token will be required to run this target.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make sendgrid-target-adapter && ./_output/sendgrid-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`           - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`         - Define the default logging configuration
  - `K_METRICS_CONFIG='''`        - Define the prometheus metrics configuration
  - `SENDGRID_API_KEY`            - SendGrid API key
  - `SENDGRID_DEFAULT_FROM_EMAIL` - (optional) Default sender email
  - `SENDGRID_DEFAULT_FROM_NAME`  - (optional) Default sender name
  - `SENDGRID_DEFAULT_TO_EMAIL`   - (optional) Default receiver email
  - `SENDGRID_DEFAULT_TO_NAME`    - (optional) Default receiver name 
  - `SENDGRID_DEFAULT_MESSAGE`    - (optional) Default message to send
  - `SENDGRID_DEFAULT_SUBJECT`    - (optional) Default subject of the email

## Creating a SendGrid Target

A full deployment example is located in the [samples](../samples/sendgrid) directory. It can be deployed via the following steps:

* Update the `100-secrets.yaml` file to include the API key.  
* Update the `200-target.yaml` file with the optional `defaultFromEmail`, `defaultFromName`, `defaultToEmail`,`defaultToName`, & `defaultSubject` parameters. 
* Apply the configuration via `kubectl`


**Note:** If there is not a default value specified for all of the optional fields, the event received by that deployment *MUST* contain all of the information noted in the [Event Types](#event-types), save **Message**, or the Target **will** **fail**

### Status

* The SendGridTarget requires only one Secret, the APIKey, to be provided. 

* A Status summary will be added to the SendGridTarget object indicating the conditions required for the target to meet.

* When ready, the `status.address.url` will point to the internal point where the CloudEvents should be sent.

### Sendgrid Target as an Event Sink

#### Once deployed, the SendGrid Target adapter is available. This means it can be used as a Sink for Knative components. 

* The included example trigger ['300-trigger.yaml'](../samples/sendgrid/300-trigger.yaml) could be applied as follows: 

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: sendgrid-sample-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: SendGridTarget
      name: triggermesh-email
```


* Modify the SinkBinding to trigger the Target instead of the Broker:
  
```yaml
[...]
  sink:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: SendGridTarget
      name: triggermesh-email # this should match the SendGrid target name
[...]
```

### Talking to the SendGrid Target

Sendgrid target can be configured with default fields for all available parameters that can be overridden at runtime using the received event's payload.

The SendGrid event Target accepts a [JSON][ce-jsonformat] payload with the following properties that will overwrite their respective `spec` parameters.

| Name  |  Type |  Comment | Required
|---|---|---|---|
| **FromName** | string | Sender's name |false |
| **FromEmail** | string | Sender's email | false |
| **ToName** | string | Recipient's name | false |
| **ToEmail** | string | Recipient's email | false |
| **Message** | string | Contents of the message body | false |
| **Subject** | string | Assigns a subject to the email | false |

When a **Message** property is **not** present, the entire cloud event is passed into the email `body` by default.

**Note:** If there is not a default value specified for all of the optional fields, the event received by that deployment *MUST* contain all of the information noted in the [Event Types](#event-types), save **Message**, or the Target **will** **fail**

### Example

An example of a Cloudevent being passed via a Curl command:

```
curl -v "localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sendgrid.email.send" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"fromEmail":"richard@triggermesh.com","toEmail":"bob@gmail.com","fromName":"richard","toName":"bob","message":"hello","subject":"Hello World"}'
```


An example email sent from the Sendgrid Target with the **Message** parameter omitted from the curl example above will look as follows:

```email
from: richard <richard@triggermesh.com>
to:	bob <bob@gmail.com>
date:	Sep 12, 2020, 12:41 AM
subject: Hello World

Validation: valid Context Attributes, specversion: 1.0 type: dev.knative.samples.helloworld source: dev.knative.samples/helloworldsource id: 536808d3-88be-4077-9d7a-a3f162705f79 time: 2020-09-12T04:41:00.000610299Z datacontenttype: application/json Extensions, knativearrivaltime: 2020-09-12T04:41:00.006331845Z knativehistory: default-kne-trigger-kn-channel.midimansland.svc.cluster.local Data, { "FromEmail":"richard@triggermesh.com","ToEmail":"bob@gmail.com", \
         "FromName":"richard","ToName":"bob","Subject":"Hello World" } 
```

[ce-jsonformat]: https://github.com/cloudevents/spec/blob/v1.0/json-format.md
