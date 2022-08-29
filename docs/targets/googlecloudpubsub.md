# Event Target for Google Cloud Pub/Sub

This event target receives [CloudEvents][ce] over HTTP and sends them to a pre-defined Google Cloud Pub Sub Topic.


## Prerequisite(s)

- Google Cloud Console account.
- A service account and it's associated JSON credentials.
- a pre-existing Google Cloud Pub Sub Topic.


## Deploying an Instance of the Target

1. [Clone][clone] the Triggermesh mono repo and navigate to the example manifest located in the [`triggermesh/config/samples/targets/googlecloudpubsub/`][sample-manifest] section of the Triggermesh mono repo.

1. Replace the example credentials in the `100-secret.yaml` file with the JSON credentials of a valid Service Account.

1. Replace the empty `topic` field in the `200-target.yaml` file with the name of the Google Cloud Pub Sub Topic to which you want to send events.

1. Apply the manifest to a running Triggermesh instance.
```
kubectl apply -f triggermesh/config/samples/targets/googlecloudpubsub/
```

## Event Types
### Arbitrary
This target consumes events of any type.

[ce]: https://cloudevents.io/
[ce-jsonformat]: https://github.com/cloudevents/spec/blob/v1.0/json-format.md
[sample-manifest]: https://github.com/triggermesh/triggermesh/tree/main/config/samples/targets/googlecloudpubsub
[clone]: https://github.com/triggermesh/triggermesh/archive/refs/heads/main.zip
