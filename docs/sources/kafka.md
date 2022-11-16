# Kafka event source

This event source acts as a consumer of a Kafka Cluster and forwards all messages it receives
as CloudEvents'.

## Contents

- [Kafka event source](#kafka-event-source)
  - [Prerequisites](#prerequisites)
  - [Creating a KafkaSource](#creating-a-kafka-source)
    - [SASL-PLAIN](#with-sasl-plain)
    - [Kerberos-SSL](#with-kerberos-ssl)
  - [Status](#status)

## Prerequisites

A running Kafka cluster.

## Creating a KafkaSource

### SASL-PLAIN

This section demonstrates how to configure a KafkaSource to use SASL-PLAIN authentication.

```yaml
apiVersion: sources.triggermesh.io/v1alpha1
kind: KafkaSource
metadata:
  name: sample
spec:
  groupID: test-consumer-group
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
  sink:
    ref:
      apiVersion: eventing.knative.dev/v1
      kind: Broker
      name: default
```

### Kerberos-SSL

This section demonstrates how to configure a KafkaSource to use Kerberos-SSL authentication.

Before creating the `KafkaSource`, we are going to create some secrets that the `KafkaSource` will need for the authentication with Kerberos + SSL.

- The kerberos config file.
- The kerberos keytab file.
- The CA Cert file.

```console
kubectl create secret generic config --from-file=krb5.conf
kubectl create secret generic keytab --from-file=krb5.keytab
kubectl create secret generic cacert --from-file=ca-cert.pem
```

```yaml
apiVersion: sources.triggermesh.io/v1alpha1
kind: KafkaSource
metadata:
  name: sample
spec:
  groupID: test-consumer-group
  bootstrapServers:
    - kafka.example.com:9093
  topic: test-topic
  auth:
    saslEnable: true
    tlsEnable: true
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
  sink:
    ref:
      apiVersion: eventing.knative.dev/v1
      kind: Broker
      name: default
```

In order to configure the adapter correctly the following fields are mandatory:

- `boostrapservers`
- `topic`
- `salsEnable`

### Status

KafkaSource requires secrets to be provided for the credentials. Once they are present it will start. Controller
logs and events can provide detailed information about the process. A Status
summary is added to the KafkaSource object informing of the all conditions
that the source needs.

When ready, the `status.ready` will be **True**.

```console
kubectl get kafkasource
NAME     READY   REASON   URL   SINK                                                                              AGE
sample   True                   http://broker-ingress.knative-eventing.svc.cluster.local/default/default   33s
```
