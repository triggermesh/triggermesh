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

package transformation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
)

// Handler contains Pipelines for CE transformations and CloudEvents client.
type Handler struct {
	ContextPipeline *Pipeline
	DataPipeline    *Pipeline

	client cloudevents.Client
}

// ceContext represents CloudEvents context structure but with exported Extensions.
type ceContext struct {
	*cloudevents.EventContextV1 `json:",inline"`
	Extensions                  map[string]interface{} `json:"Extensions,omitempty"`
}

// NewHandler creates Handler instance.
func NewHandler(context, data []v1alpha1.Transform) (Handler, error) {
	contextPipeline, err := newPipeline(context)
	if err != nil {
		return Handler{}, err
	}

	dataPipeline, err := newPipeline(data)
	if err != nil {
		return Handler{}, err
	}

	sharedVars := storage.New()
	contextPipeline.setStorage(sharedVars)
	dataPipeline.setStorage(sharedVars)

	ceClient, err := cloudevents.NewClientHTTP()
	if err != nil {
		return Handler{}, err
	}

	return Handler{
		ContextPipeline: contextPipeline,
		DataPipeline:    dataPipeline,

		client: ceClient,
	}, nil
}

// Start runs CloudEvent receiver and applies transformation Pipeline
// on incoming events.
func (t *Handler) Start(ctx context.Context, sink string) error {
	log.Println("Starting CloudEvent receiver")
	var receiver interface{}
	receiver = t.receiveAndReply
	if sink != "" {
		ctx = cloudevents.ContextWithTarget(ctx, sink)
		receiver = t.receiveAndSend
	}

	return t.client.StartReceiver(ctx, receiver)
}

func (t *Handler) receiveAndReply(event cloudevents.Event) (*cloudevents.Event, error) {
	return t.applyTransformations(event)
}

func (t *Handler) receiveAndSend(ctx context.Context, event cloudevents.Event) error {
	result, err := t.applyTransformations(event)
	if err != nil {
		return err
	}
	return t.client.Send(ctx, *result)
}

func (t *Handler) applyTransformations(event cloudevents.Event) (*cloudevents.Event, error) {
	log.Printf("Received %q event", event.Type())
	// HTTPTargets sets content type from HTTP headers, i.e.:
	// "datacontenttype: application/json; charset=utf-8"
	// so we must use "contains" instead of strict equality
	if !strings.Contains(event.DataContentType(), cloudevents.ApplicationJSON) {
		log.Printf("CE Content Type %q is not supported", event.DataContentType())
		return nil, fmt.Errorf("CE Content Type %q is not supported", event.DataContentType())
	}

	localContext := ceContext{
		EventContextV1: event.Context.AsV1(),
		Extensions:     event.Context.AsV1().GetExtensions(),
	}

	localContextBytes, err := json.Marshal(localContext)
	if err != nil {
		log.Printf("Cannot encode CE context: %v", err)
		return nil, fmt.Errorf("cannot encode CE context: %w", err)
	}

	// Run init step such as load Pipeline variables first
	t.ContextPipeline.initStep(localContextBytes)
	t.DataPipeline.initStep(event.Data())

	// CE Context transformation
	localContextBytes, err = t.ContextPipeline.apply(localContextBytes)
	if err != nil {
		log.Printf("Cannot apply transformation on CE context: %v", err)
		return nil, fmt.Errorf("cannot apply transformation on CE context: %w", err)
	}

	if err := json.Unmarshal(localContextBytes, &localContext); err != nil {
		log.Printf("Cannot decode CE new context: %v", err)
		return nil, fmt.Errorf("cannot decode CE new context: %w", err)
	}
	event.Context = localContext
	for k, v := range localContext.Extensions {
		if err := event.Context.SetExtension(k, v); err != nil {
			log.Printf("Cannot set CE extension: %v", err)
			return nil, fmt.Errorf("cannot set CE extension: %w", err)
		}
	}

	// CE Data transformation
	data, err := t.DataPipeline.apply(event.Data())
	if err != nil {
		log.Printf("Cannot apply transformation on CE data: %v", err)
		return nil, fmt.Errorf("cannot apply transformation on CE data: %w", err)
	}
	if err = event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		log.Printf("Cannot set data: %v", err)
		return nil, fmt.Errorf("cannot set data: %w", err)
	}

	log.Printf("Sending %q event", event.Type())
	return &event, nil
}
