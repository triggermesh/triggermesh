module github.com/triggermesh/triggermesh

go 1.15

// Top-level module control over the exact version used for important direct dependencies.
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
replace k8s.io/client-go => k8s.io/client-go v0.20.7

require (
	github.com/cloudevents/sdk-go/v2 v2.4.1
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/stretchr/testify v1.7.0
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.17.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.20.7
	knative.dev/eventing v0.24.3
	knative.dev/pkg v0.0.0-20210902173607-953af0138c75
	knative.dev/serving v0.24.1
)

require (
	cloud.google.com/go/billing v0.1.0
	cloud.google.com/go/container v0.1.0 // indirect
	cloud.google.com/go/firestore v1.1.0
	cloud.google.com/go/logging v1.4.2
	cloud.google.com/go/monitoring v0.1.0 // indirect
	cloud.google.com/go/pubsub v1.16.0
	cloud.google.com/go/storage v1.16.1
	cloud.google.com/go/trace v0.1.0 // indirect
	cloud.google.com/go/workflows v1.0.0
	github.com/Azure/azure-amqp-common-go/v3 v3.0.1
	github.com/Azure/azure-event-hubs-go/v3 v3.3.13
	github.com/Azure/azure-sdk-for-go v57.1.0+incompatible
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest v0.11.20
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/ZachtimusPrime/Go-Splunk-HTTP/splunk/v2 v2.0.1
	github.com/aliyun/aliyun-oss-go-sdk v2.1.9+incompatible
	github.com/andygrunwald/go-jira v1.13.0
	github.com/aws/aws-sdk-go v1.37.1
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beeker1121/goque v2.1.0+incompatible // indirect
	github.com/clbanning/mxj v1.8.4
	github.com/confluentinc/confluent-kafka-go v1.4.2 // indirect
	github.com/devigned/tab v0.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elastic/go-elasticsearch/v7 v7.6.0
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/hashicorp/go-uuid v1.0.1
	github.com/inconshreveable/log15 v0.0.0-20201112154412-8562bdadbbac // indirect
	github.com/jarcoal/httpmock v1.0.8
	github.com/kevinburke/go-types v0.0.0-20210723172823-2deba1f80ba7 // indirect
	github.com/kevinburke/rest v0.0.0-20210506044642-5611499aa33c // indirect
	github.com/kevinburke/twilio-go v0.0.0-20200203063821-378e630e02da
	github.com/logzio/logzio-go v0.0.0-20200316143903-ac8fc0e2910e
	github.com/nukosuke/go-zendesk v0.9.2
	github.com/oracle/oci-go-sdk v24.3.0+incompatible
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac
	github.com/sendgrid/rest v2.6.5+incompatible // indirect
	github.com/sendgrid/sendgrid-go v3.6.3+incompatible
	github.com/shirou/gopsutil v3.21.8+incompatible // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tektoncd/pipeline v0.24.1
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.2.1 // indirect
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/tools v0.1.6 // indirect
	google.golang.org/api v0.57.0
	google.golang.org/genproto v0.0.0-20210921142501-181ce0d877f6
	google.golang.org/grpc v1.40.0
	gopkg.in/confluentinc/confluent-kafka-go.v1 v1.4.2
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	knative.dev/networking v0.0.0-20210622182128-53f45d6d2cfa
)
