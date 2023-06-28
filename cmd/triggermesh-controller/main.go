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

package main

import (
	"os"

	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"

	"github.com/triggermesh/triggermesh/pkg/extensions/reconciler/function"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/jqtransformation"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/synchronizer"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/transformation"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/xmltojsontransformation"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/xslttransformation"
	"github.com/triggermesh/triggermesh/pkg/routing/reconciler/filter"
	"github.com/triggermesh/triggermesh/pkg/routing/reconciler/splitter"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscloudwatchlogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscloudwatchsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscodecommitsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscognitoidentitysource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscognitouserpoolsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awsdynamodbsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awseventbridgesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awskinesissource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awsperformanceinsightssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awss3source"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awssnssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awssqssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureactivitylogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureblobstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureeventgridsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureeventhubssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureiothubsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azurequeuestoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebusqueuesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebussource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebustopicsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/cloudeventssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudauditlogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudbillingsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudpubsubsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudsourcerepositoriessource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/httppollersource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/ibmmqsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/kafkasource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/mongodbsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/ocimetricssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/salesforcesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/slacksource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/solacesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/twiliosource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/webhooksource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/zendesksource"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awscomprehendtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awsdynamodbtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awseventbridgetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awskinesistarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awslambdatarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awss3target"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awssnstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awssqstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/azureeventhubstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/azuresentineltarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/azureservicebustarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/cloudeventstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/datadogtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/elasticsearchtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudfirestoretarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudpubsubtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudstoragetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudworkflowstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlesheettarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/httptarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/ibmmqtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/jiratarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/kafkatarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/logzmetricstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/logztarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/mongodbtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/oracletarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/salesforcetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/sendgridtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/slacktarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/solacetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/splunktarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/twiliotarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/zendesktarget"
)

func main() {
	ctx := signals.NewContext()

	if namespace, set := os.LookupEnv("WORKING_NAMESPACE"); set {
		ctx = injection.WithNamespaceScope(ctx, namespace)
	}

	sharedmain.MainWithContext(ctx, "triggermesh-controller",
		// sources
		awscloudwatchlogssource.NewController,
		awscloudwatchsource.NewController,
		awscodecommitsource.NewController,
		awscognitoidentitysource.NewController,
		awscognitouserpoolsource.NewController,
		awsdynamodbsource.NewController,
		awseventbridgesource.NewController,
		awskinesissource.NewController,
		awsperformanceinsightssource.NewController,
		awss3source.NewController,
		awssnssource.NewController,
		awssqssource.NewController,
		azureactivitylogssource.NewController,
		azureblobstoragesource.NewController,
		azureeventgridsource.NewController,
		azureeventhubssource.NewController,
		azureiothubsource.NewController,
		azurequeuestoragesource.NewController,
		azureservicebusqueuesource.NewController,
		azureservicebussource.NewController,
		azureservicebustopicsource.NewController,
		cloudeventssource.NewController,
		googlecloudauditlogssource.NewController,
		googlecloudbillingsource.NewController,
		googlecloudpubsubsource.NewController,
		googlecloudsourcerepositoriessource.NewController,
		googlecloudstoragesource.NewController,
		httppollersource.NewController,
		ibmmqsource.NewController,
		kafkasource.NewController,
		mongodbsource.NewController,
		ocimetricssource.NewController,
		salesforcesource.NewController,
		slacksource.NewController,
		solacesource.NewController,
		twiliosource.NewController,
		webhooksource.NewController,
		zendesksource.NewController,
		// targets
		awscomprehendtarget.NewController,
		awsdynamodbtarget.NewController,
		awseventbridgetarget.NewController,
		awskinesistarget.NewController,
		awslambdatarget.NewController,
		awss3target.NewController,
		awssnstarget.NewController,
		awssqstarget.NewController,
		azureeventhubstarget.NewController,
		azuresentineltarget.NewController,
		azureservicebustarget.NewController,
		cloudeventstarget.NewController,
		elasticsearchtarget.NewController,
		googlecloudstoragetarget.NewController,
		googlecloudfirestoretarget.NewController,
		googlecloudworkflowstarget.NewController,
		googlecloudpubsubtarget.NewController,
		googlesheettarget.NewController,
		httptarget.NewController,
		ibmmqtarget.NewController,
		datadogtarget.NewController,
		jiratarget.NewController,
		kafkatarget.NewController,
		logztarget.NewController,
		logzmetricstarget.NewController,
		mongodbtarget.NewController,
		oracletarget.NewController,
		salesforcetarget.NewController,
		sendgridtarget.NewController,
		slacktarget.NewController,
		solacetarget.NewController,
		splunktarget.NewController,
		twiliotarget.NewController,
		zendesktarget.NewController,
		// flow
		jqtransformation.NewController,
		synchronizer.NewController,
		transformation.NewController,
		xmltojsontransformation.NewController,
		xslttransformation.NewController,
		// extensions
		function.NewController,
		// routing
		filter.NewController,
		splitter.NewController,
	)
}
