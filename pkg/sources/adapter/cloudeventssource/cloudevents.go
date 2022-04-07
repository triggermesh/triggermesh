/*
Copyright 2021 TriggerMesh Inc.

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

package cloudeventssource

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"go.uber.org/zap"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

type cloudEventsHandler struct {
	basicAuths []v1alpha1.HTTPBasicAuth
	tokens     []v1alpha1.HTTPToken
	// corsAllowOrigin string

	ceServer cloudevents.Client
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Start implements adapter.Adapter.
func (h *cloudEventsHandler) Start(ctx context.Context) error {
	return h.ceServer.StartReceiver(ctx, h.handle)
}

func (h *cloudEventsHandler) handle(ctx context.Context, e event.Event) protocol.Result {
	if result := h.ceClient.Send(ctx, e); !cloudevents.IsACK(result) {
		h.logger.Error("could not send CloudEvent")
		return protocol.ResultNACK
	}

	return protocol.ResultACK
}
