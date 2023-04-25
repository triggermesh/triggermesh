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

package azureeventhubssource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

func TestProcessEventGridMessage(t *testing.T) {
	testData := fakeEventGridCloudEvent()

	msgPrcsr := &eventGridMessageProcessor{}
	events, err := msgPrcsr.Process(testData)

	require.NoError(t, err)
	assert.Len(t, events, 3)

	assert.Equal(t, "Microsoft.Storage.BlobCreated", events[0].Type())
	assert.Equal(t, "ImagePushed", events[1].Type())
	assert.Equal(t, "Microsoft.EventHub.CaptureFileCreated", events[2].Type())
}

// fakeEventGridCloudEvent returns a eventhub.Event which payload represents an
// event from Event Grid using the CloudEvent schema.
func fakeEventGridCloudEvent() *azeventhubs.ReceivedEventData {
	return &azeventhubs.ReceivedEventData{
		EventData: azeventhubs.EventData{
			Body: sampleEventGridEvent,
		},
	}
}

// Event Grid event containing multiple sample payloads from
// * https://docs.microsoft.com/en-us/azure/event-grid/event-schema-blob-storage
// * https://docs.microsoft.com/en-us/azure/event-grid/event-schema-container-registry
// * https://docs.microsoft.com/en-us/azure/event-grid/event-schema-event-hubs
var sampleEventGridEvent = []byte(`[
{
  "source": "/subscriptions/{subscription-id}/resourceGroups/Storage/providers/Microsoft.Storage/storageAccounts/my-storage-account",
  "subject": "/blobServices/default/containers/test-container/blobs/new-file.txt",
  "type": "Microsoft.Storage.BlobCreated",
  "time": "2017-06-26T18:41:00.9584103Z",
  "id": "831e1650-001e-001b-66ab-eeb76e069631",
  "data": {
    "api": "PutBlockList",
    "clientRequestId": "6d79dbfb-0e37-4fc4-981f-442c9ca65760",
    "requestId": "831e1650-001e-001b-66ab-eeb76e000000",
    "eTag": "\"0x8D4BCC2E4835CD0\"",
    "contentType": "text/plain",
    "contentLength": 524288,
    "blobType": "BlockBlob",
    "url": "https://my-storage-account.blob.core.windows.net/testcontainer/new-file.txt",
    "sequencer": "00000000000004420000000000028963",
    "storageDiagnostics": {
      "batchId": "b68529f3-68cd-4744-baa4-3c0498ec19f0"
    }
  },
  "specversion": "1.0"
},
{
  "id": "831e1650-001e-001b-66ab-eeb76e069631",
  "source": "/subscriptions/<subscription-id>/resourceGroups/<resource-group-name>/providers/Microsoft.ContainerRegistry/registries/<name>",
  "subject": "aci-helloworld:v1",
  "type": "ImagePushed",
  "time": "2018-04-25T21:39:47.6549614Z",
  "data": {
    "id": "31c51664-e5bd-416a-a5df-e5206bc47ed0",
    "timestamp": "2018-04-25T21:39:47.276585742Z",
    "action": "push",
    "target": {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 3023,
      "digest": "sha256:213bbc182920ab41e18edc2001e06abcca6735d87782d9cef68abd83941cf0e5",
      "length": 3023,
      "repository": "aci-helloworld",
      "tag": "v1"
    },
    "request": {
      "id": "7c66f28b-de19-40a4-821c-6f5f6c0003a4",
      "host": "demo.azurecr.io",
      "method": "PUT",
      "useragent": "docker/18.03.0-ce go/go1.9.4 git-commit/0520e24 os/windows arch/amd64 UpstreamClient(Docker-Client/18.03.0-ce \\\\(windows\\\\))"
    }
  },
  "specversion": "1.0"
},
{
  "source": "/subscriptions/<guid>/resourcegroups/rgDataMigrationSample/providers/Microsoft.EventHub/namespaces/tfdatamigratens",
  "subject": "eventhubs/hubdatamigration",
  "type": "Microsoft.EventHub.CaptureFileCreated",
  "time": "2017-08-31T19:12:46.0498024Z",
  "id": "14e87d03-6fbf-4bb2-9a21-92bd1281f247",
  "data": {
    "fileUrl": "https://tf0831datamigrate.blob.core.windows.net/windturbinecapture/tfdatamigratens/hubdatamigration/1/2017/08/31/19/11/45.avro",
    "fileType": "AzureBlockBlob",
    "partitionId": "1",
    "sizeInBytes": 249168,
    "eventCount": 1500,
    "firstSequenceNumber": 2400,
    "lastSequenceNumber": 3899,
    "firstEnqueueTime": "2017-08-31T19:12:14.674Z",
    "lastEnqueueTime": "2017-08-31T19:12:44.309Z"
  },
  "specversion": "1.0"
}
]`)
