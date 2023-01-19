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

package splitter

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/eventing/pkg/utils"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"

	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/routing/v1alpha1/splitter"
	routinglisters "github.com/triggermesh/triggermesh/pkg/client/generated/listers/routing/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/routing/adapter/common/env"
)

const serverPort int = 8080

// Handler parses Cloud Events, determines if they pass a filter, and sends them to a subscriber.
type Handler struct {
	// receiver receives incoming HTTP requests
	receiver *kncloudevents.HTTPMessageReceiver
	// sender sends requests to downstream services
	sender *kncloudevents.HTTPMessageSender

	splitterLister routinglisters.SplitterNamespaceLister
	logger         *zap.SugaredLogger
}

// NewEnvConfig satisfies env.ConfigConstructor.
func NewEnvConfig() env.ConfigAccessor {
	return &env.Config{}
}

// NewAdapter returns a constructor for the source's adapter.
func NewAdapter(string) pkgadapter.AdapterConstructor {
	return func(ctx context.Context, _ pkgadapter.EnvConfigAccessor, _ cloudevents.Client) pkgadapter.Adapter {
		logger := logging.FromContext(ctx)

		sender, err := kncloudevents.NewHTTPMessageSenderWithTarget("")
		if err != nil {
			logger.Panicf("failed to create message sender: %v", err)
		}

		informer := informerv1alpha1.Get(ctx)
		ns := injection.GetNamespaceScope(ctx)

		return &Handler{
			receiver:       kncloudevents.NewHTTPMessageReceiver(serverPort),
			sender:         sender,
			splitterLister: informer.Lister().Splitters(ns),
			logger:         logger,
		}
	}
}

// Start begins to receive messages for the handler.
//
// HTTP POST requests to the root path (/) are accepted.
//
// This method will block until ctx is done.
func (h *Handler) Start(ctx context.Context) error {
	return h.receiver.StartListen(ctx, h)
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	splitter, err := parseRequestURI(request.URL.Path)
	if err != nil {
		h.logger.Errorw("Unable to parse path as splitter", zap.Error(err), zap.String("path", request.RequestURI))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := request.Context()

	message := cehttp.NewMessageFromHttpRequest(request)
	// cannot be err, but makes linter complain about missing err check
	//nolint
	defer message.Finish(nil)

	event, err := binding.ToEvent(ctx, message)
	if err != nil {
		h.logger.Errorw("Failed to extract event from request", zap.Error(err))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Debugw("Received message", zap.Any("splitter", splitter))

	s, err := h.splitterLister.Get(splitter)
	if err != nil {
		h.logger.Errorw("Unable to get the Splitter", zap.Error(err))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	for i, e := range h.split(s.Spec.Path, event) {
		e.SetID(fmt.Sprintf("%s-%d", event.ID(), i))
		e.SetType(s.Spec.CEContext.Type)
		e.SetSource(s.Spec.CEContext.Source)
		for key, value := range s.Spec.CEContext.Extensions {
			e.SetExtension(key, value)
		}
		// we may want to keep responses and send them back to the source
		_, err := h.sendEvent(ctx, request.Header, s.Status.SinkURI.String(), e)
		if err != nil {
			h.logger.Errorw("Failed to send the event", zap.Error(err))
		}
	}

	writer.WriteHeader(http.StatusOK)
}

func (h *Handler) split(path string, e *event.Event) []*event.Event {
	val := gjson.Get(string(e.Data()), path)
	if !val.IsArray() {
		val = gjson.Parse("[" + val.Raw + "]")
	}

	var result []*event.Event
	for _, v := range val.Array() {
		newCE := cloudevents.NewEvent()
		if err := newCE.SetData(cloudevents.ApplicationJSON, []byte(v.Raw)); err != nil {
			h.logger.Errorw("Failed to set event data", zap.Error(err))
			continue
		}
		newCE.DataBase64 = false
		result = append(result, &newCE)
	}
	return result
}

func (h *Handler) sendEvent(ctx context.Context, headers http.Header, target string, event *cloudevents.Event) (*http.Response, error) {
	// Send the event to the subscriber
	req, err := h.sender.NewCloudEventRequestWithTarget(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create the request: %w", err)
	}

	message := binding.ToMessage(event)
	// cannot be err, but makes linter complain about missing err check
	//nolint
	defer message.Finish(nil)

	additionalHeaders := utils.PassThroughHeaders(headers)
	err = kncloudevents.WriteHTTPRequestWithAdditionalHeaders(ctx, message, req, additionalHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	resp, err := h.sender.Send(req)
	if err != nil {
		err = fmt.Errorf("failed to dispatch message: %w", err)
	}

	return resp, err
}

func parseRequestURI(path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("incorrect number of parts in the path, expected 2, actual %d, '%s'", len(parts), path)
	}
	return parts[2], nil
}
