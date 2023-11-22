# GoogleSheet event target for Knative Eventing

This event target integrates with Google Sheet, using received Cloud Event message's to append information to a 
Google Sheet document. 

## Contents

- [GoogleSheet event target for Knative Eventing](#googlesheet-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Creating a GoogleSheet Target](#creating-a-googlesheet-target)
    - [Status](#status)
    - [GoogleSheet Target as an Event Sink](#googlesheet-target-as-an-event-sink)
    - [Sending Messages to the GoogleSheet Target](#sending-messages-to-the-googlesheet-target)

## Prerequisites

A GoogleSheet API key is required to utilize this target. The steps to obtain a
key are outlined in the [Sheets API Docs](https://developers.google.com/sheets/api/quickstart/go).

Use the `client_email` field within the credentials JSON file you downloaded
to ***SHARE the Google Spreadsheets*** you want the Target to have access to.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make googlesheet-target-adapater && ./_output/googlesheet-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`       - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`     - Define the default logging configuration
  - `K_METRICS_CONFIG='''`    - Define the prometheus metrics configuration
  - `SHEET_ID`                - The Google Sheet ID to write the event details to
  - `GOOGLE_CREDENTIALS_JSON` - JWT Token to authenticate against Google's service
  - `DEFAULT_SHEET_PREFIX`    - Sheet prefix to write the event to

## Creating a GoogleSheet Target

Once the GoogleSheet Target Controller has been deployed, create an integration
by adding a GoogleSheetTarget like the following:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: GoogleSheetTarget
metadata:
  name: triggermesh-googlesheet
spec:
#Below is an example Spreadsheet ID. Change this 
  id: 14GKZKWVB2TsYy31cCZ43YwA1LoOlVeL4nB7jlZbgFAk
# Static prefix assignment for reciving CloudEvents without prior transformation
  defaultPrefix: <Default Prefix>
#These values should not change
  googleServiceAccount:
    secretKeyRef:
      name: googlesheet
      key: credentials
```

`id,` `defaultprefix,` &  `credentials` are all REQUIRED to deploy.

`id` is a unique identifier that can be retrieved from the URL path or parameters:
  - from path: `https://docs.google.com/spreadsheets/d/<SHEET_ID>/edit`
  - from query string: `https://docs.google.com/spreadsheet/ccc?key=<SHEET_ID>`
  
A full deployment example is located in the [samples directory](../samples/googlesheet)

### Status

A GoogleSheetTarget requires a single secret : `credentials.` Once it is present the Target will create a Knative Service. Controller logs and events can provide detailed information about the process. A Status summary is added to the GoogleSheetTarget object informing of the all conditions that the target needs.

When ready the `status.address.url` will point to the internal point where Cloud Events should be sent.

### GoogleSheet Target as an Event Sink

A GoogleSheet Target is addressable, which means it can be used as a Sink for Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: googlesheet-sample-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: GoogleSheetTarget
      name: triggermesh-googlesheet
```

A sample sink binding to a GoogleSheet Target deployment. 

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
      kind: GoogleSheetTarget
      name: triggermesh-googlesheet
```

### Sending Messages to the GoogleSheet Target

 A GoogleSheet Target will, by default, accept any CloudEvent and pass the entire event into a new row on the specified Sheet.

#### Example Sending Aribtrary Events
```sh
curl -v localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: some.message.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"mgs":"Hello from TriggerMesh using GoogleSheet!"}'
```

### Sending Events of Type `io.triggermesh.googlesheet.append`
The GoogleSheet event Target accepts a [JSON][ce-jsonformat] payload with the following properties:

| Name  |  Type |  Comment | Required
|---|---|---|---|
| **sheet_name** | string | The name of the sheet to create and or populate |true |
| **rows** | []string | List of data to populate the new row.  | true |


```sh
curl -v localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: io.triggermesh.googlesheet.append" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"rows":["Hello from TriggerMesh using GoogleSheet!", "test","sheet1"],"sheet_name":"Sheet1"}'
```

