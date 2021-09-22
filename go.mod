module github.com/triggermesh/triggermesh

go 1.15

// Top-level module control over the exact version used for important direct dependencies.
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
replace k8s.io/client-go => k8s.io/client-go v0.19.7

require (
	github.com/cloudevents/sdk-go/v2 v2.5.0
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/stretchr/testify v1.7.0
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.19.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.19.7
	knative.dev/eventing v0.22.1
	knative.dev/pkg v0.0.0-20210331065221-952fdd90dbb0
	knative.dev/serving v0.22.0
)

require github.com/aws/aws-sdk-go v1.37.1

require (
	cloud.google.com/go v0.72.0
	cloud.google.com/go/firestore v1.1.0
	cloud.google.com/go/storage v1.10.0
	contrib.go.opencensus.io/exporter/prometheus v0.4.0 // indirect
	github.com/ZachtimusPrime/Go-Splunk-HTTP/splunk/v2 v2.0.2
	github.com/aliyun/aliyun-oss-go-sdk v2.1.10+incompatible
	github.com/andygrunwald/go-jira v1.14.0
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/clbanning/mxj v1.8.4
	github.com/confluentinc/confluent-kafka-go v1.7.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elastic/go-elasticsearch/v7 v7.14.0
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/hashicorp/go-uuid v1.0.1
	github.com/jarcoal/httpmock v1.0.8
	github.com/kevinburke/go-types v0.0.0-20210723172823-2deba1f80ba7 // indirect
	github.com/kevinburke/rest v0.0.0-20210506044642-5611499aa33c // indirect
	github.com/kevinburke/twilio-go v0.0.0-20210327194925-1623146bcf73
	github.com/logzio/logzio-go v1.0.2
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/nukosuke/go-zendesk v0.9.2
	github.com/oracle/oci-go-sdk v24.3.0+incompatible
	github.com/prometheus/common v0.30.0 // indirect
	github.com/robertkrimen/otto v0.0.0-20210614181706-373ff5438452
	github.com/sendgrid/rest v2.6.4+incompatible // indirect
	github.com/sendgrid/sendgrid-go v3.10.0+incompatible
	github.com/tektoncd/pipeline v0.25.0
	github.com/ttacon/builder v0.0.0-20170518171403-c099f663e1c2 // indirect
	github.com/ttacon/libphonenumber v1.2.1 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20210903162142-ad29c8ab022f // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/tools v0.1.5 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20210416161957-9910b6c460de
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/confluentinc/confluent-kafka-go.v1 v1.7.0
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	k8s.io/gengo v0.0.0-20210203185629-de9496dff47b // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	knative.dev/networking v0.0.0-20210331064822-999a7708876c
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)
