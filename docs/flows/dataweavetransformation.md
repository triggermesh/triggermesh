steps to test locally:

export DW_SPELL="output application/json --- payload filter (item) -> item.age > 15"



curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sendgrid.email.send" \
       -H "Ce-Source: dev.knative.samples/helloworldsource" \
       -H "Content-Type: application/json" \
       -d 'items[{"age": "20"}]'



        echo '[{"name": "User1","age": 19},{"name": "User2","age": 18},{"name": "User3","age": 15},{"name": "User4","age": 13},{"name": "User5","age": 16}]' | ./dw "output application/json --- payload filter (item) -> item.age > 17"