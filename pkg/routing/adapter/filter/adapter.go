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

package filter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/eventing/pkg/utils"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/routing/v1alpha1/filter"
	routinglisters "github.com/triggermesh/triggermesh/pkg/client/generated/listers/routing/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/routing/adapter/common/env"
	"github.com/triggermesh/triggermesh/pkg/routing/eventfilter"
	"github.com/triggermesh/triggermesh/pkg/routing/eventfilter/cel"
)

const serverPort int = 8080

// Handler parses Cloud Events, determines if they pass a filter, and sends them to a subscriber.
type Handler struct {
	// receiver receives incoming HTTP requests
	receiver *kncloudevents.HTTPMessageReceiver
	// sender sends requests to downstream services
	sender *kncloudevents.HTTPMessageSender

	filterLister routinglisters.FilterNamespaceLister
	logger       *zap.SugaredLogger

	// expressions is the map of trigger refs with precompiled CEL expressions
	// TODO (tzununbekov): Add cleanup
	expressions *expressionStorage
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
			receiver:     kncloudevents.NewHTTPMessageReceiver(serverPort),
			sender:       sender,
			filterLister: informer.Lister().Filters(ns),
			logger:       logger,

			expressions: newExpressionStorage(),
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

	filter, err := parseRequestURI(request.URL.Path)
	if err != nil {
		h.logger.Errorw("Unable to parse path as filter", zap.Error(err), zap.String("path", request.RequestURI))
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

	h.logger.Debug("Received message", zap.Any("filter", filter))

	f, err := h.filterLister.Get(filter)
	if err != nil {
		h.logger.Errorw("Unable to get the Filter", zap.Error(err))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	cond, exists := h.expressions.get(f.UID, f.Generation)
	if !exists {
		cond, err = cel.CompileExpression(f.Spec.Expression)
		if err != nil {
			h.logger.Errorw("Failed to compile filter expression", zap.Error(err), zap.Any("filter", filter))
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		h.expressions.set(f.UID, f.Generation, cond)
	}

	filterResult := filterEvent(ctx, cond, *event)
	if filterResult == eventfilter.FailFilter {
		return
	}

	event = updateAttributes(f.Status, event)
	h.send(ctx, writer, request.Header, f.Status.SinkURI.String(), event)
}

func updateAttributes(fs commonv1alpha1.Status, event *event.Event) *event.Event {
	if len(fs.CloudEventAttributes) == 1 {
		event.SetType(fs.CloudEventAttributes[0].Type)
		event.SetSource(fs.CloudEventAttributes[0].Source)
	}
	return event
}

func (h *Handler) send(ctx context.Context, writer http.ResponseWriter, headers http.Header, target string, event *cloudevents.Event) {
	// send the event to trigger's subscriber
	response, err := h.sendEvent(ctx, headers, target, event)
	if err != nil {
		h.logger.Errorw("Failed to send event", zap.Error(err))
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Successfully dispatched message", zap.Any("target", target))

	// If there is an event in the response write it to the response
	_, err = h.writeResponse(ctx, writer, response, target)
	if err != nil {
		h.logger.Errorw("Failed to write response", zap.Error(err))
	}
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

// The return values are the status
func (h *Handler) writeResponse(ctx context.Context, writer http.ResponseWriter, resp *http.Response, target string) (int, error) {
	response := cehttp.NewMessageFromHttpResponse(resp)
	// cannot be err, but makes linter complain about missing err check
	//nolint
	defer response.Finish(nil)

	if response.ReadEncoding() == binding.EncodingUnknown {
		// Response doesn't have a ce-specversion header nor a content-type matching a cloudevent event format
		// Just read a byte out of the reader to see if it's non-empty, we don't care what it is,
		// just that it is not empty. This means there was a response and it's not valid, so treat
		// as delivery failure.
		body := make([]byte, 1)
		n, _ := response.BodyReader.Read(body)
		response.BodyReader.Close()
		if n != 0 {
			// Note that we could just use StatusInternalServerError, but to distinguish
			// between the failure cases, we use a different code here.
			writer.WriteHeader(http.StatusBadGateway)
			return http.StatusBadGateway, errors.New("received a non-empty response not recognized as CloudEvent. The response MUST be or empty or a valid CloudEvent")
		}
		h.logger.Debug("Response doesn't contain a CloudEvent, replying with an empty response", zap.Any("target", target))
		writer.WriteHeader(resp.StatusCode)
		return resp.StatusCode, nil
	}

	event, err := binding.ToEvent(ctx, response)
	if err != nil {
		// Like in the above case, we could just use StatusInternalServerError, but to distinguish
		// between the failure cases, we use a different code here.
		writer.WriteHeader(http.StatusBadGateway)
		// Malformed event, reply with err
		return http.StatusBadGateway, err
	}

	eventResponse := binding.ToMessage(event)
	// cannot be err, but makes linter complain about missing err check
	//nolint
	defer eventResponse.Finish(nil)

	if err := cehttp.WriteResponseWriter(ctx, eventResponse, resp.StatusCode, writer); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to write response event: %w", err)
	}

	h.logger.Debug("Replied with a CloudEvent response", zap.Any("target", target))

	return resp.StatusCode, nil
}

func filterEvent(ctx context.Context, filter cel.ConditionalFilter, event cloudevents.Event) eventfilter.FilterResult {
	var filters eventfilter.Filters
	if filter.Expression != nil {
		filters = append(filters, &filter)
	}

	return filters.Filter(ctx, event)
}

func parseRequestURI(path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("incorrect number of parts in the path, expected 2, actual %d, '%s'", len(parts), path)
	}
	return parts[2], nil
}
