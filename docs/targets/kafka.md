# Kafka event target

This event target integrates with Kafka, where any Cloud Event received are published to a Kafka Cluster.

## Contents

- [Kafka event target](#kafka-event-target)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Creating a KafkaTarget](#creating-a-kafkatarget)
    - [SASL-PLAIN](#sasl-plain)
    - [Kerberos-SSL](#kerberos-ssl)
  - [Status](#status)
    - [KafkaTarget as an event Sink](#kafkatarget-as-an-event-sink)
    - [Sending messages to the KafkaTarget](#sending-messages-to-the-kafkatarget)

## Prerequisites

A running Kafka cluster.

## Creating a KafkaTarget

### SASL-PLAIN

This section demonstrates how to configure a KafkaTarget to use SASL-PLAIN authentication.

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: KafkaTarget
metadata:
  name: sample
spec:
  bootstrapServers:
  - kafka.example.com:9092
  topic: test-topic
  auth:
    saslEnable: true
    tlsEnable: false
    securityMechanism: PLAIN
    username: admin
    password:
      value: admin-secret
```

### Kerberos-SSL

This section demonstrates how to configure a KafkaTarget to use Kerberos-SSL authentication.

Before creating the `KafkaTarget`, we are going to create some secrets that the `KafkaTarget` will need for the authentication with Kerberos + SSL.

- The kerberos config file.
- The kerberos keytab file.
- The CA Cert file.

```console
kubectl create secret generic config --from-file=krb5.conf
kubectl create secret generic keytab --from-file=krb5.keytab
kubectl create secret generic cacert --from-file=ca-cert.pem
```

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: KafkaTarget
metadata:
  name: sample
spec:
  saslEnable: true
  tlsEnable: true
  bootstrapServers:
  - kafka.example.com:9093
  topic: test-topic
  securityMechanism: GSSAPI
  kerberosAuth:
    username: kafka
    kerberosRealm: EXAMPLE.COM
    kerberosServiceName: kafka
    kerberosConfig:
      valueFromSecret:
        name: config
        key: krb5.conf
    kerberosKeytab:
      valueFromSecret:
        name: keytab
        key: krb5.keytab
  sslAuth:
    insecureSkipVerify: true
    sslCA:
      valueFromSecret:
        name: cacert
        key: ca-cert
```

 In order to configure the adapter correctly the following fields are mandatory:

- boostrapservers
- topic
- saslEnable

## Status

KafkaTarget requires secrets to be provided for the credentials. Once they are present it will create a Knative Service. Controller
logs and events can provide detailed information about the process. A Status
summary is added to the KafkaTarget object informing of the all conditions
that the target needs.

When ready the `status.address.url` will point to the internal point where Cloud Events should be sent.

```console
kubectl get kafkatarget
NAME     URL                                                   READY   REASON   AGE
sample   http://kafkatarget-sample.default.svc.cluster.local   True             1m
```

### KafkaTarget as an event Sink

A `KafkaTarget` is addressable, which means it can be used as a Sink for Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: kafka-sample-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: KafkaTarget
      name: triggermesh-kafka
```

- A sample sink binding to a `KafkaTarget` deployment.

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
      kind: KafkaTarget
      name: triggermesh-kafka
```

### Sending messages to the KafkaTarget

 A KafkaTarget will, by default, accept any CloudEvent and pass the entire event into a message body.

`curl` can be used from a container in the cluster pointing to the `KafkaTarget` exposed URL:

```console
curl -v http://kafkatarget-int1-9fg4abc7d44bdd0204bd0a221bea9453k.default.svc.cluster.local
 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: some.message.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 739481h1-34gt-9801-4h0d-g6e048192l23" \
 -d '{"message":"Hello from TriggerMesh using Kafka!"}'
```
