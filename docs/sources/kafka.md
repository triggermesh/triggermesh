# Kafka event source

This event source acts as a consumer of a Kafka events and forwards all messages it receives
as CloudEvents'.

## Contents

- [Kafka event source](#kafka-event-source)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Creating a Kafka Source](#creating-a-kafka-source)
      + [SASL-PLAIN](#with-sasl-plain)
      + [Kerberos-SSL](#with-kerberos-ssl)
    - [Status](#status)

## Prerequisites

A running Kafka cluster.

## Creating a Kafka Source with SASL-PLAIN

```yaml
apiVersion: sources.triggermesh.io/v1alpha1
kind: KafkaSource
metadata:
  name: sample
spec:
  groupID: test-consumer-group
  bootstrapServers:
    - kafka.example.com:9092
  topics: 
    - test-topic
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

## Creating a Kafka Source with Kerberos-SSL

Before creating the `KafkaSource`, we are going to create some secrets that the `KafkaSource` will need for the authentication with Kerberos + SSL.

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
  topics: 
    - test-topic
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
- `topics`
- `salsEnable`
- `tlsEnable`

### Status

KafkaSource requires secrets to be provided for the credentials. Once they are present it will start. Controller
logs and events can provide detailed information about the process. A Status
summary is added to the KafkaSource object informing of the all conditions
that the source needs.
