
```
curl -v localhost:8080\
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: any.event.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"message":"Hello from TriggerMesh!"}'
```


export NAMESPACE=se
export K_LOGGING_CONFIG=[]
export K_METRICS_CONFIG=[]
export OSS_ENDPOINT=
export OSS_BUCKET=
export OSS_ACCESS_KEY_SECRET=
export OSS_ACCESS_KEY_ID=

