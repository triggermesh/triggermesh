curl -v "10.1.215.232:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sendgrid.email.send" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"fromEmail":"I LOVE YOU"}'

Response: 
```
{"positive":0.999502420425415,"negative":0.00006757258961442858,"mixed":0.00005553230221266858,"result":"Positive"}
```

curl -v "localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sendgrid.email.send" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"fromEmail":"I LOVE YOU", "other":"you are great", "another":"awesome job!"}'

Response: 
```
{"positive":2.979724109172821,"negative":0.001508750458015129,"mixed":0.004781584390002536,"result":"Positive"}
```

curl -v "localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sendgrid.email.send" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d '{"fromEmail":"you suck", "other":"hate you", "another":"go to hell"}'

Response: 
```
{"positive":0.05191964528057724,"negative":2.70785391330719,"mixed":0.08987980522215366,"result":"Negative"}
```


export NAMESPACE=default

export K_LOGGING_CONFIG=''

export K_METRICS_CONFIG=''

export AWS_ACCESS_KEY_ID=

export AWS_SECRET_ACCESS_KEY=

export COMPREHEND_REGION=us-west-2

export COMPREHEND_LANGUAGE=en