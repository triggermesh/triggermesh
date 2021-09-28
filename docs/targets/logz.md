

## Sending information to logz

```cmd
curl -v http://logztarget-tmlogz.logz.svc.cluster.local \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: any.event.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"message":"Hello from TriggerMesh using GoogleSheet!"}'
 ```

export NAMESPACE=se
export K_LOGGING_CONFIG=[]
export K_METRICS_CONFIG=[]
export LOGZ_SHIPPING_TOKEN=
export LOGZ_LISTENER_URL=listener.logz.io


## Logz.io Documentation

https://docs.logz.io/shipping/log-sources/go.html