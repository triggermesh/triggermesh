Steps to debug locally:

1:
```
export QUERY=".foo | .."
```

2:
```
go run cmd/jqtransformation-adapter/main.go
```

3:
```
curl -v "localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sendgrid.email.send" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"foo":"richard@triggermesh.com"}'
```

4:
EXPECTED RESULT:
```
* Connected to localhost (::1) port 8080 (#0)
> POST / HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.77.0
> Accept: */*
> Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79
> Ce-Specversion: 1.0
> Ce-Type: io.triggermesh.sendgrid.email.send
> Ce-Source: dev.knative.samples/helloworldsource
> Content-Type: application/json
> Content-Length: 33
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79
< Ce-Source: dev.knative.samples/helloworldsource
< Ce-Specversion: 1.0
< Ce-Time: 2022-01-24T18:10:40.953971Z
< Ce-Type: io.triggermesh.sendgrid.email.send
< Content-Length: 25
< Content-Type: application/json
< Date: Mon, 24 Jan 2022 18:10:40 GMT
< 
* Connection #0 to host localhost left intact
"richard@triggermesh.com"
```