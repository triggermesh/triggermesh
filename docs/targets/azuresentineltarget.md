

You can send any request that the standard Azure Sentinel API [Incidents - Create Or Update](https://docs.microsoft.com/en-us/rest/api/securityinsights/stable/incidents/create-or-update) supports. IE we are expecting this [Request Body](https://docs.microsoft.com/en-us/rest/api/securityinsights/stable/incidents/create-or-update#request-body) at the payload of the events.

Note there are some minimum required fields:
```
properties.severity
properties.status
properties.title
```

Example supported event:
```
curl -v http://azuresentineltarget-hello-sentinel.default.10.107.122.79.sslip.io\
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: any.event.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"etag": "some-etag", "properties": {"providerIncidentId": "12", "status":"new", "severity": "high", "title": "some-title", "description": "some-description", "owner":{"assignedTo": "some-owner"},"additionalData": {"alertProductNames": ["some-product","some-other-product"]}}}'
```


export to run locally:

```
export NAMESPACE=se
export K_LOGGING_CONFIG=[]
export K_METRICS_CONFIG=[]
export AZURE_SUBSCRIPTION_ID=
export AZURE_RESOURCE_GROUP=
export AZURE_WORKSPACE=
export AZURE_CLIENT_SECRET=
export AZURE_CLIENT_ID=
export AZURE_TENANT_ID=

```
