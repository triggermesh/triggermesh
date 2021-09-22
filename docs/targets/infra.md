# Infrastructure Event Target for Knative Eventing

This event target extends CloudEvents flows with advanced manipulation of events and stateful management.

## Contents

- [Infrastructure Event Target for Knative Eventing](#infrastructure-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Infra Target features](#infra-target-features)
    - [Advanced transformation](#advanced-transformation)
    - [State header management](#state-header-management)
  - [Deploying from Code](#deploying-from-code)
  - [Create Infra Target Integration](#create-infra-target-integration)

## Infra Target features

The Infra integration needs to be planned for the features it offers:

- Advanced transformation: a javascript snippet that can use the `input` variable to access the incoming CloudEvent. The returned value will be used as the response CloudEvent. The script will be halted if it exceeds the configured timeout.

- Stateful tracking: a stateful brige's first event should be consumed at this target to be added the stateful headers that should be propagated throuhout the rest of the bridge's flow.

- Stateful storage: not implemented yet.

### Advanced transformation

Infra target allows users to transform events in almost any way with by droping a few javascript lines of code.

- The user's snippet should rely on the `input` variable which contains the CloudEvent received at the target.
- Response event should be returned from the code using the `return` keyword.
- Nil can be returned, in which case there wont be an event response.
- Javascript runtime is ES5 based but not complete.
- The code will be halted if the [timeout](#deploying-from-code) is reached (two seconds by default)
- It is important when returning an event that the `event.type` for the outgoing event does not match the incoming one to avoid hot loops.
- If ID is not found for the outgoing event an arbitrary unique ID will be set.

The `input` parameter is structured by dumping a CloudEvent object into it:

```js
// full cloud event object
input

// header fields
input.id
input.type
input.source

// header extensions
input.myextension

// input data
input.data

// input data fields
input.data.user.lastlogin

// if a field contains a non valid variable name in it's path
// it can be accessed by indexer
input.data.message["date-sent"]
```

Returned event follows the same rules as the `input` variable. If the intended output CloudEvent is a slight modification on the incoming one, the `input` variable can be modified and returned:

```js
// avoid same type loops
input.type = "modified.type"
// remove a field we do not want at the output event
delete input.data["address"]

return intput
```

When a new JSON schema is needed for the outgoing event, it is preferred to create a new one and copy the `input` variables we need.

```js
nevent = {
  "source": input.source,
  "type": "my.new.type",
  "data": {"username": input.data.user[0].name, "text": input.data.message}
}

return nevent
```

Although the code does not run on a fully featured Javascript environment, most of the functionality is available to build events

```js
if (input.data.notification.type == "sensible") {
  // on sensible information skip response events
  return nil
}

nevent = {
  "source": input.source,
  "type": "my.new.type",
  "data": {
    "ingested": Date().toString(),
    "message": input.data.
  }
}

nevent.data.channel = ( input.data.user.group == "support" ) ? "tickets" : "devhelp"

return nevent
```

### State header management

The state header can have 3 values:

- `none`: no action will be taken regarding CloudEvent headers
- `ensure`: if `statefulbridge` header is not found it will be set with the bridge name as a value . If `statefulid` header is not found it will be set with an arbitrary ID.
- `propagate`: if headers `statefulbridge`, `statefulid` or `statestep` are found in the incoming event, they will be copied to the outgoing event.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly. From the top-level source directory:

```sh
make infra-target-adapater && ./_output/infra-target-adapter
```

Several environment variables will need to be set prior to invoking the adapter such as:

- `NAMESPACE` at Kubernetes where the adapter is being run. Mandatory.
- `K_LOGGING_CONFIG` logging configuration as defined at Knative. Mandatory
- `K_METRICS_CONFIG` metrics configuration as defined at Knative. Mandatory
- `INFRA_STATE_HEADERS_POLICY` policy for state header creation and propagation. .Defaults to `propagate`
- `INFRA_STATE_BRIDGE` bridge name where this component runs. Mandatory if headers policy is set to `ensure`
- `INFRA_SCRIPT_CODE` javascript code snippet to be executed. Optional
- `INFRA_SCRIPT_TIMEOUT` number of milliseconds before the script execution is halted. Defaults to 2 seconds.
- `INFRA_TYPE_LOOP_PROTECTION` errors if incoming and outgoing types match. Defaults to true.

## Create Infra Target Integration

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: InfraTarget
metadata:
  name: zd-to-slack
  namespace: mynamespace
spec:
  script:
    code:  |-
      event = {"channel":"C01112A09FT", "text": input.data.user + ": " + input.data.message}
      return event
    timeout: 1000
  state:
    headersPolicy: propagate
    bridge: zendesk-to-slack
  typeLoopProtection: true

```

Fields at the `spec` above match the environment variables listed for the adapter:

- `script.code` javascript code snippet to be executed. Optional
- `script.timeout` number of milliseconds before the script execution is halted. Defaults to 2000.
- `state.headersPolicy` policy for state header creation and propagation. Defaults to `propagate`
- `state.bridge` bridge name where this component runs. Mandatory if headers policy is set to `ensure`.
- `infraLoopProtection` will fail and return a nil event if incoming and outgoing types are the same. Defaults to true.
