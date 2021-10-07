# Sending arbitrary events
The target will accept arbitrary events and use the Event ID as the Document name
```
curl -v "http://googlecloudfirestoretarget-googlecloudfirestore.dmo.svc.cluster.local" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9dsdf7a-a3f162s705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.arbitrary" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"data":"hello World"}'
```

# Sending events of type io.triggermesh.google.firestore.write
If it is preferd to specify the collection on each call to the target, an event of type `io.triggermesh.google.firestore.write` can be sent.
The payload body must contain the following attributes:
 `collection` : Defines the firebase collection to be written under
 `document` : Defines the firebase document name to be written
 `data` : Defines the items to be written to the document

```
curl -v "http://broker-ingress.knative-eventing.svc.cluster.local/dmo/default" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.google.firestore.write" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"collection":"eventtst","document":"doctests1","data":{"fromEmail":"bob@triggermesh.com","hello":"pls"}}'
```

# Sending events of type io.triggermesh.google.firestore.query.tables
Return all tables in a provided collection
```
curl -v "http://broker-ingress.knative-eventing.svc.cluster.local/dmo/default" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.google.firestore.query.tables" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"collection":"eventtst"}'
```

# Sending events of type io.triggermesh.google.firestore.query.table
Return a selected table from a collection 
```
curl -v "http://broker-ingress.knative-eventing.svc.cluster.local/dmo/default" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.google.firestore.query.table" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"collection":"deploydemo","document":"536808d3-88be-4077-9d7a-a3f162s705f79"}'
```

# Running locally 
export NAMESPACE=se
export K_LOGGING_CONFIG=[]
export K_METRICS_CONFIG=[]
export DISCARD_CE_CONTEXT=true
export GOOGLE_FIRESTORE_DEFAULT_COLLECTION="defaultcol"
export GOOGLE_FIRESTORE_PROJECT_ID=
export GOOGLE_CREDENTIALS_JSON=''
