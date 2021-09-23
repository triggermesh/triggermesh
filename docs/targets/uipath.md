### "io.triggermesh.uipath.job"

#### Example starting job with 'InputArguments'   
```
curl -v -X POST http://localhost:8080  \
    -H "content-type: application/json"  \
    -H "ce-specversion: 1.0"  \
    -H "ce-source: curl-pablo"  \
    -H "ce-type: io.triggermesh.uipath.job.start"  \
    -H "ce-id: 123-abc" \
    -H "ce-statefulid: my-stateful-12345" \
    -H "ce-somethingelse: hello-world" \
    -H "statefulid: hello-world" \
    -d '{"InputArguments":"{\"AccountName__c\":\"Trigger\",\"CompanyName__c\":\"Messh\",\"PhoneNumber__c\":\"919-3092-1021\",\"Discount__c\":\"1\",\"AccountID__c\":\"Mesj\",\"FullAddress__c\":\"123 Fiction St. Raleigh\",\"Description__c\":\"Goat Burger Patty\",\"Email__c\":\"jeff@triggermesh.com\",\"Quantity__c\":\"3\",\"UnitPrice__c\":\"123\"}"}'
```
#### Example starting a job with no arguments.
```
curl -v -X POST http://localhost:8080   \
    -H "content-type: application/json"  \
    -H "ce-specversion: 1.0"  \
    -H "ce-source: curl-pablo"  \
    -H "ce-type: io.triggermesh.uipath.job.start"  \
    -H "ce-id: 123-abc" \
    -H "ce-statefulid: my-stateful-12345" \
    -H "ce-somethingelse: hello-world" \
    -H "statefulid: hello-world" \
    -d '{}'
```

### "io.triggermesh.uipath.queue.post"

The specified queue must be pre-existing within the UiPath Organization ID (defined via the `UIPATH_ORGANIZATION_UNIT_ID` enviorment variable or `organizationUnitID` in the spec.)
```
curl -v -X POST http://10.1.67.221:8080  \
    -H "content-type: application/json"  \
    -H "ce-specversion: 1.0"  \
    -H "ce-source: curl-pablo"  \
    -H "ce-type: io.triggermesh.uipath.queue.post"  \
    -H "ce-id: 123-abc" \
    -d '{"Name":"qtst", "Priority":"Normal", "SpecificContent": {"Bob":"the build" , "big":"pog"}}'
```

### LOCAL ENV
In order to test locally provide the following enviorment variables and execute `go run cmd/uipath-target-adapter/main.go`


export UIPATH_ROBOT_NAME=<ROBOT_NAME>
export UIPATH_PROCESS_NAME=<PROCESS_NAME>
export UIPATH_TENANT_NAME=<TENANT_NAME>
export UIPATH_ACCOUNT_LOGICAL_NAME=<LOGICAL_NAME>
export UIPATH_CLIENT_ID=<CLIENT_ID>
export UIPATH_USER_KEY=<USER_KEY>
export UIPATH_ORGANIZATION_UNIT_ID=<ORGANIZATION_UNIT_ID>
export NAMESPACE=<NAMESPACE>
export K_METRICS_CONFIG=
export K_LOGGING_CONFIG=
