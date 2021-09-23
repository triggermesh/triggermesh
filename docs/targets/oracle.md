# Oracle Cloud Knative Target

The Oracle Cloud Knative Target acts as a target frontend to Oracle Cloud related services.
At this time, only [Oracle Functions](https://docs.cloud.oracle.com/en-us/iaas/Content/Functions/Concepts/functionsoverview.htm) (or Fn) is supported.

## Prerequisites (Global)

Regardless of what event targets exist currently or in the future, the following is required:
  - Tenancy OCID where the target components reside
  - User OCID with access to the target components
  - User API access tokens:
    - API Private Key
    - Private Key's passphrase
    - Fingerprint of API key
  - Oracle Cloud Region where the component resides

Setting up an account on the Oracle Cloud and obtaining the prerequisite data is outside the
scope of this readme, but obtaining most of the prerequisite data can be found in the
[Oracle Developer Resources](https://docs.cloud.oracle.com/en-us/iaas/Content/Functions/Tasks/functionssetupapikey.htm)
### Oracle Function Requirements

To invoke an Oracle Cloud Function, the target function's OCID is required.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make oracle-target-adapater && ./_output/oracle-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`                  - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`                - Define the default logging configuration
  - `K_METRICS_CONFIG='''`               - Define the prometheus metrics configuration
  - `ORACLE_API_PRIVATE_KEY`             - User's private API key as x509 key
  - `ORACLE_API_PRIVATE_KEY_PASSPHRASE`  - Passphrase to decript the private API key
  - `ORACLE_API_PRIVATE_KEY_FINGERPRINT` - Fingerprint of the private API key
  - `USER_OCID`                          - Oracle Cloud ID associated with the user's key 
  - `TENANT_OCID`                        - Oracle Cloud ID associated with the tenant
  - `ORACLE_REGION`                      - Oracle Cloud region where the target service resides
  - `ORACLE_FUNCTION`                    - Oracle Cloud ID associated with the function being invoked 

A full deployment example is located in the [samples](../samples/oracle) directory

## Creating an Oracle Service Secret

To access any of the Oracle Cloud services, an API private key will be required.  This is
stored in a secret along with the passphrase to decrypt the key, and a fingerprint of the
key associated with the user.

A sample secret would resemble:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oraclecreds
type: Opaque
stringData:
  apiPassphrase: ""
  apiKeyFingerprint: "94:d4:c0:2f:6a:18:85:1a:31:a1:85:69:d5:47:fc:5d"
  apiKey: |-
    -----BEGIN RSA PRIVATE KEY-----
    MIIEogIBAAKCAQEAwRapSZ6+4wS18BkCu70Ic0IMeFksVsIJKZ+8xIZfMeGpW2zn
    [...]
    -----END RSA PRIVATE KEY-----

```

## Creating an Oracle Cloud Target

The target spec consists of the global parameters as a part of the core Spec, and a
sub spec for each service.

An example of a target for a function would resemble the following:
```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: OracleTarget
metadata:
  name: triggermesh-oracle-function
spec:
  oracleApiPrivateKey:
    secretKeyRef:
      name: oraclecreds
      key: apiKey
  oracleApiPrivateKeyPassphrase:
    secretKeyRef:
      name: oraclecreds
      key: apiPassphrase
  oracleApiPrivateKeyFingerprint:
    secretKeyRef:
      name: oraclecreds
      key: apiKeyFingerprint
  oracleTenancy: ocid1.tenancy.oc1..aaaaaaaaav23f45mqyxmwu4x3s2uhuh4rb2bwdpgb5kbpjqvwiiqufhsq6za
  oracleUser: ocid1.user.oc1..aaaaaaaacaxtveoy4zx7rsg7lanexmouxjxay6godthrfsocpl6ggrfpbiuq
  oracleRegion: us-phoenix-1
  function:
    function: ocid1.fnfunc.oc1.phx.aaaaaaaaaajrgy4on66e6krko73h2im5qaiiagecg5hmbcqib2kpbzlcy3bq
```

The thing to note with the example above is, with the exception of `oracleRegion`, the attributes
require the Oracle Cloud ID (OCID) values for the various attributes including the function.

## Triggering the Oracle Cloud Event

The triggering mechanism needs to be put in place to listen for the event, and trigger
the service:

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: oracle-cloud-function-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: OracleTarget
      name: triggermesh-oracle-function
```

In the case of Functions, an event is created as a response that can be published and acted upon.

To invoke the target, a sample cURL command can be used (assuming minikube is used and a tunnel has been established):

```console
curl -v http://oracletarget-triggermesh-oracle-function.default.example.com \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: dev.knative.source.oracle" \
 -H "Ce-Source: dev.knative.source.oracle" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"name": "bob"}'

* Rebuilt URL to: http://oracletarget-triggermesh-oracle-function.default.example.com/
*   Trying 10.99.31.116...
* TCP_NODELAY set
* Connected to oracletarget-triggermesh-oracle-function.default.example.com (10.99.31.116) port 80 (#0)
> POST / HTTP/1.1
> Host: oracletarget-triggermesh-oracle-function.default.example.com
> User-Agent: curl/7.58.0
> Accept: */*
> Content-Type: application/json
> Ce-Specversion: 1.0
> Ce-Type: dev.knative.source.oracle
> Ce-Source: dev.knative.source.oracle
> Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79
> Content-Length: 15
> 
* upload completely sent off: 15 out of 15 bytes
< HTTP/1.1 200 OK
< ce-id: 6f8cbbc3-01b0-48d3-8f12-d3d5558ad6b9
< ce-source: ocid1.fnapp.oc1.phx.aaaaaaaaaehdhsmharxvyp4pvnsgsnd35am5u7ckjzivwmsmove37eckjika
< ce-specversion: 1.0
< ce-subject: ocid1.fnfunc.oc1.phx.aaaaaaaaaajrgy4on66e6krko73h2im5qaiiagecg5hmbcqib2kpbzlcy3bq
< ce-time: 2020-06-03T01:21:26.126325681Z
< ce-type: functions.oracletargets.targets.triggermesh.io
< content-length: 29
< content-type: application/json
< date: Wed, 03 Jun 2020 01:21:25 GMT
< x-envoy-upstream-service-time: 27294
< server: envoy
< 
* Connection #0 to host oracletarget-triggermesh-oracle-function.default.example.com left intact
{"processed":{"name": "bob"}} 
```

