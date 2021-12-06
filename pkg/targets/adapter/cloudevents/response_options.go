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

package cloudevents

import (
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

// Optional headers
const (
	StatefulWorkflowHeader         = "statefulbridge"
	StatefulWorkflowInstanceHeader = "statefulid"

	ProcessedTypeHeader   = "processedtype"
	ProcessedSourceHeader = "processedsource"
	ProcessedIDHeader     = "processedid"
)

// EventResponseOption given the incoming and outgoing event,
// modifies the response event before being sent.
type EventResponseOption func(in, out *cloudevents.Event) error

// ResponseWithSubject is an option for modifying returned event subject.
func ResponseWithSubject(subject string) EventResponseOption {
	return func(in, out *cloudevents.Event) error {
		return out.Context.SetSubject(subject)
	}
}

// ResponseWithID is an option for modifying returned event ID.
func ResponseWithID(ID string) EventResponseOption {
	return func(in, out *cloudevents.Event) error {
		return out.Context.SetID(ID)
	}
}

// ResponseWithStatefulHeaders creates stateful headers if not present.
func ResponseWithStatefulHeaders(bridge string) EventResponseOption {
	return func(in, out *cloudevents.Event) error {
		ext := in.Context.GetExtensions()

		b, ok := ext[StatefulWorkflowHeader]
		if !ok {
			b = bridge
		}
		if err := out.Context.SetExtension(StatefulWorkflowHeader, b); err != nil {
			return fmt.Errorf("error ensuring %q extension: %w", StatefulWorkflowHeader, err)
		}

		instance, ok := ext[StatefulWorkflowInstanceHeader]
		if !ok {
			instance = uuid.New().String()
		}
		if err := out.Context.SetExtension(StatefulWorkflowInstanceHeader, instance); err != nil {
			return fmt.Errorf("error ensuring %q extension: %w", StatefulWorkflowInstanceHeader, err)
		}
		return nil
	}
}

// ResponseWithProcessedHeaders creates processed headers, propagating
// information about the incoming event headers into the outgoing event.
func ResponseWithProcessedHeaders() EventResponseOption {
	return func(in, out *cloudevents.Event) error {

		if err := out.Context.SetExtension(ProcessedTypeHeader, in.Context.GetType()); err != nil {
			return fmt.Errorf("error setting %q extension: %w", ProcessedTypeHeader, err)
		}

		if err := out.Context.SetExtension(ProcessedSourceHeader, in.Context.GetSource()); err != nil {
			return fmt.Errorf("error setting %q extension: %w", ProcessedSourceHeader, err)
		}

		if err := out.Context.SetExtension(ProcessedIDHeader, in.Context.GetID()); err != nil {
			return fmt.Errorf("error setting %q extension: %w", ProcessedIDHeader, err)
		}

		return nil
	}
}

// ResponseWithDataContentType is an option for modifying returned event content type.
func ResponseWithDataContentType(dct string) EventResponseOption {
	return func(in, out *cloudevents.Event) error {
		return out.Context.SetDataContentType(dct)
	}
}
