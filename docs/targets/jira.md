# Jira Knative Target

This event target integrates with Jira, using received CloudEvent messages to
create and retrieve Jira tickets or perform custom actions using Jira API.

## Contents

- [Jira Knative Target](#jira-knative-target)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Creating a Jira Token Secret](#creating-a-jira-token-secret)
  - [Creating a Jira Target](#creating-a-jira-target)
    - [Sending Messages to the Jira Target](#sending-messages-to-the-jira-target)

## Prerequisites

1. Jira instance or Atlassian cloud tenant.
1. User API token.

To create the user API token at Jira:

- Open the Account settings > Security > [Create and manage API Tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
- Press `Create API token` and fill the token name.
- Copy the API token and create a secret for the Jira token at TriggerMesh.
## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make jira-target-adapater && ./_output/jira-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`          - Usually set by the kubernetes cluster.
  - `K_LOGGING_CONFIG=''`        - Define the default logging configuration.
  - `K_METRICS_CONFIG='''`       - Define the prometheus metrics configuration.
  - `JIRA_AUTH_USER`             - Jira user.
  - `JIRA_AUTH_TOKEN`            - Jira API token.
  - `JIRA_URL`                   - Jira server URL.

A full deployment example is located in the [samples](../samples/jira) directory

## Creating a Jira Token Secret

To access the Jira services, an API private key will be required.

A sample secret would resemble:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: jiratoken
type: Opaque
stringData:
  token: "jira-api-token"
```

## Creating a Jira Target

An example of a Jira target for a function would resemble the following:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: JiraTarget
metadata:
  name: tmjira
spec:
  auth:
    user: woodford@triggermesh.com
    token:
      secretKeyRef:
        name: jira
        key: token
  url: https://tmtest.atlassian.net
```

Jira fields `user`, `token` and `url` are required.

### Sending Messages to the Jira Target

The Jira target accepts these event types:

- `com.jira.issue.create`

The Jira target will create an issue when receiving this event type.

```sh
curl -v -X POST http://jiratarget-tmjira.default.svc.cluster.local \
-H "content-type: application/json" \
-H "ce-specversion: 1.0" \
-H "ce-source: curl-triggermesh" \
-H "ce-type: io.triggermesh.jira.issue.create" \
-H "ce-id: 123-abc" \
-d '{
    "fields": {
       "project":
       {
          "key": "IP"
       },
       "labels": ["alpha","beta"],
       "summary": "Day 30.",
       "description": "Issue created using TriggerMesh Jira Target",
       "issuetype": {
          "name": "Task"
       },
       "assignee": {
          "accountId": "5fe0704c9edf280075f188f0"
       }
   }
}'
```

- `com.jira.issue.get`

The Jira target will retrieve an issue when receiving this event type.

```sh
curl -v -X POST http://jiratarget-tmjira.default.svc.cluster.local \
-H "content-type: application/json" \
-H "ce-specversion: 1.0" \
-H "ce-source: curl-triggermesh" \
-H "ce-type: io.triggermesh.jira.issue.get" \
-H "ce-id: 123-abc" \
-d '{"id":"IP-9"}'
```

- `com.jira.custom`

The Jira target will request the Jira API when this event type is received. The CloudEvent data expects a generic API request as seen at this example:

```sh
curl -v -X POST http://jiratarget-tmjira.default.svc.cluster.local \
-H "content-type: application/json" \
-H "ce-specversion: 1.0" \
-H "ce-source: curl-triggermesh" \
-H "ce-type: io.triggermesh.jira.custom" \
-H "ce-id: 123-abc" \
-d '{
    "method": "GET",
    "path": "/rest/api/3/user/assignable/search",
    "query": {"project": "IP"}
   }'
```

Please, refer to the [Jira API](https://developer.atlassian.com/cloud/jira/software/rest/intro/) on how to fill in values for these requests.

