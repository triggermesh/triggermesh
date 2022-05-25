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

package sources

import (
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awscodecommit"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awscognitouserpool"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awsdynamodb"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awseventbridge"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awskinesis"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awssns"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/awssqs"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureactivitylogs"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureblobstorage"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureeventgrid"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureeventhubs"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureiothub"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azurequeuestorage"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureservicebusqueue"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/azureservicebustopic"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/googlecloudauditlogs"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/googlecloudpubsub"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/googlecloudsourcerepositories"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/googlecloudstorage"
	_ "github.com/triggermesh/triggermesh/test/e2e/sources/webhook"
)
