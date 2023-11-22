
## Sending information to Google Workflows

### io.trigermesh.google.workflows.run

```cmd
curl -v http://googlecloudworkflowstarget-googlecloudworkflows.dmo.svc.cluster.local \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: io.trigermesh.google.workflows.run" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"parent":"projects/ultra-hologram-297914/locations/us-central1/workflows/demowf","executionName":"projects/ultra-hologram-297914/locations/us-central1/workflows/demowf/executions/testex"}'
 ```

export NAMESPACE=se
export K_LOGGING_CONFIG=[]
export K_METRICS_CONFIG=[]
export GOOGLE_WORKFLOWS_CREDENTIALS_JSON=
