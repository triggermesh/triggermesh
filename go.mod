module github.com/triggermesh/triggermesh

go 1.17

// Knative and CloudEvents are the common denominator to all TriggerMesh components.
require (
	github.com/cloudevents/sdk-go/v2 v2.6.1
	knative.dev/eventing v0.26.0
	knative.dev/pkg v0.0.0-20210919202233-5ae482141474
	knative.dev/serving v0.26.0
)

// Top-level module control over the exact version used for important direct dependencies.
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
replace k8s.io/client-go => k8s.io/client-go v0.21.4

require (
	cloud.google.com/go/billing v0.1.0
	cloud.google.com/go/firestore v1.1.0
	cloud.google.com/go/logging v1.4.2
	cloud.google.com/go/pubsub v1.17.0
	cloud.google.com/go/storage v1.18.2
	cloud.google.com/go/workflows v1.0.0
	github.com/Azure/azure-event-hubs-go/v3 v3.3.16
	github.com/Azure/azure-sdk-for-go v58.3.0+incompatible
	github.com/Azure/azure-service-bus-go v0.11.3
	github.com/Azure/azure-storage-queue-go v0.0.0-20191125232315-636801874cdd
	github.com/Azure/go-autorest/autorest v0.11.21
	github.com/Azure/go-autorest/autorest/adal v0.9.16
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/ZachtimusPrime/Go-Splunk-HTTP/splunk/v2 v2.0.2
	github.com/aliyun/aliyun-oss-go-sdk v2.1.9+incompatible
	github.com/amenzhinsky/iothub v0.8.0
	github.com/andygrunwald/go-jira v1.14.0
	github.com/aws/aws-sdk-go v1.37.1
	github.com/devigned/tab v0.1.1
	github.com/elastic/go-elasticsearch/v7 v7.15.1
	github.com/google/cel-go v0.9.0
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-uuid v1.0.1
	github.com/jarcoal/httpmock v1.0.8
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kevinburke/twilio-go v0.0.0-20200203063821-378e630e02da
	github.com/logzio/logzio-go v0.0.0-20200316143903-ac8fc0e2910e
	github.com/nukosuke/go-zendesk v0.9.2
	github.com/oracle/oci-go-sdk v24.3.0+incompatible
	github.com/sendgrid/sendgrid-go v3.6.3+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.24.1
	github.com/tidwall/gjson v1.6.8
	go.opencensus.io v0.23.0
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.25.0
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/sdk/metric v0.24.0
	go.uber.org/zap v1.19.1
	golang.org/x/net v0.0.0-20211020060615-d418f374d309
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	google.golang.org/api v0.59.0
	google.golang.org/genproto v0.0.0-20211016002631-37fc39342514
	google.golang.org/grpc v1.41.0
	gopkg.in/confluentinc/confluent-kafka-go.v1 v1.7.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.21.4
)

require (
	cloud.google.com/go v0.97.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/cloudevents/sdk-go/observability/opencensus/v2 v2.6.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/gobuffalo/flect v0.2.3 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.opentelemetry.io/otel/sdk v1.0.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/automaxprocs v1.4.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/apiextensions-apiserver v0.21.4 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
