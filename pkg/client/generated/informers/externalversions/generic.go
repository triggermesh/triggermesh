/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	flowv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	routingv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=extensions.triggermesh.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("functions"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1alpha1().Functions().Informer()}, nil

		// Group=flow.triggermesh.io, Version=v1alpha1
	case flowv1alpha1.SchemeGroupVersion.WithResource("jqtransformations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Flow().V1alpha1().JQTransformations().Informer()}, nil
	case flowv1alpha1.SchemeGroupVersion.WithResource("synchronizers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Flow().V1alpha1().Synchronizers().Informer()}, nil
	case flowv1alpha1.SchemeGroupVersion.WithResource("transformations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Flow().V1alpha1().Transformations().Informer()}, nil
	case flowv1alpha1.SchemeGroupVersion.WithResource("xmltojsontransformations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Flow().V1alpha1().XMLToJSONTransformations().Informer()}, nil
	case flowv1alpha1.SchemeGroupVersion.WithResource("xslttransformations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Flow().V1alpha1().XSLTTransformations().Informer()}, nil

		// Group=routing.triggermesh.io, Version=v1alpha1
	case routingv1alpha1.SchemeGroupVersion.WithResource("filters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Routing().V1alpha1().Filters().Informer()}, nil
	case routingv1alpha1.SchemeGroupVersion.WithResource("splitters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Routing().V1alpha1().Splitters().Informer()}, nil

		// Group=sources.triggermesh.io, Version=v1alpha1
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awscloudwatchlogssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSCloudWatchLogsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awscloudwatchsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSCloudWatchSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awscodecommitsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSCodeCommitSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awscognitoidentitysources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSCognitoIdentitySources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awscognitouserpoolsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSCognitoUserPoolSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awsdynamodbsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSDynamoDBSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awseventbridgesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSEventBridgeSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awskinesissources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSKinesisSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awsperformanceinsightssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSPerformanceInsightsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awss3sources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSS3Sources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awssnssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSSNSSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("awssqssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AWSSQSSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureactivitylogssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureActivityLogsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureblobstoragesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureBlobStorageSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureeventgridsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureEventGridSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureeventhubssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureEventHubsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureiothubsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureIOTHubSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azurequeuestoragesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureQueueStorageSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureservicebusqueuesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureServiceBusQueueSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureservicebussources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureServiceBusSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("azureservicebustopicsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().AzureServiceBusTopicSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("cloudeventssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().CloudEventsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("googlecloudauditlogssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().GoogleCloudAuditLogsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("googlecloudbillingsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().GoogleCloudBillingSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("googlecloudpubsubsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().GoogleCloudPubSubSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("googlecloudsourcerepositoriessources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().GoogleCloudSourceRepositoriesSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("googlecloudstoragesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().GoogleCloudStorageSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("httppollersources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().HTTPPollerSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("ibmmqsources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().IBMMQSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("kafkasources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().KafkaSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("ocimetricssources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().OCIMetricsSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("salesforcesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().SalesforceSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("slacksources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().SlackSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("solacesources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().SolaceSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("twiliosources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().TwilioSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("webhooksources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().WebhookSources().Informer()}, nil
	case sourcesv1alpha1.SchemeGroupVersion.WithResource("zendesksources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Sources().V1alpha1().ZendeskSources().Informer()}, nil

		// Group=targets.triggermesh.io, Version=v1alpha1
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awscomprehendtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSComprehendTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awsdynamodbtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSDynamoDBTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awseventbridgetargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSEventBridgeTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awskinesistargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSKinesisTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awslambdatargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSLambdaTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awss3targets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSS3Targets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awssnstargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSSNSTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("awssqstargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AWSSQSTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("azureeventhubstargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AzureEventHubsTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("azuresentineltargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AzureSentinelTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("azureservicebustargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().AzureServiceBusTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("cloudeventstargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().CloudEventsTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("datadogtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().DatadogTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("elasticsearchtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().ElasticsearchTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("googlecloudfirestoretargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().GoogleCloudFirestoreTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("googlecloudpubsubtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().GoogleCloudPubSubTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("googlecloudstoragetargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().GoogleCloudStorageTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("googlecloudworkflowstargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().GoogleCloudWorkflowsTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("googlesheettargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().GoogleSheetTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("httptargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().HTTPTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("ibmmqtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().IBMMQTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("jiratargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().JiraTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("kafkatargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().KafkaTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("logzmetricstargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().LogzMetricsTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("logztargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().LogzTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("mongodbtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().MongoDBTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("oracletargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().OracleTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("salesforcetargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().SalesforceTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("sendgridtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().SendGridTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("slacktargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().SlackTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("solacetargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().SolaceTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("splunktargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().SplunkTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("twiliotargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().TwilioTargets().Informer()}, nil
	case targetsv1alpha1.SchemeGroupVersion.WithResource("zendesktargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Targets().V1alpha1().ZendeskTargets().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
