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

package bridges

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event/datacodec"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/serving"
	revisionnames "knative.dev/serving/pkg/reconciler/revision/resources/names"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
)

const (
	eventDisplayName           = "event-display"
	eventDisplayContainerImage = "gcr.io/knative-releases/knative.dev/eventing/cmd/event_display"
)

// CreateEventDisplaySink creates an event-display event sink and returns it as
// a duckv1.Destination.
func CreateEventDisplaySink(c clientset.Interface, namespace string) *duckv1.Destination {
	const internalPort uint16 = 8080
	const exposedPort uint16 = 80

	_, svc := apps.CreateSimpleApplication(c, namespace,
		eventDisplayName, eventDisplayContainerImage, internalPort, exposedPort, nil,
		apps.WithStartupProbe("/healthz"),
	)

	svcGVK := corev1.SchemeGroupVersion.WithKind("Service")

	return &duckv1.Destination{
		Ref: &duckv1.KReference{
			APIVersion: svcGVK.GroupVersion().String(),
			Kind:       svcGVK.Kind,
			Name:       svc.Name,
		},
	}
}

// EventDisplayDeploymentName returns the name of the Deployment object
// managing the event-display application, assuming that Deployment is managed
// by a Knative Service with the expected default name.
func EventDisplayDeploymentName(c dynamic.Interface, namespace string) string {
	ksvcGVR := serving.ServicesResource.WithVersion("v1")

	ksvc, err := c.Resource(ksvcGVR).Namespace(namespace).Get(context.Background(), eventDisplayName, metav1.GetOptions{})
	if err != nil {
		framework.FailfWithOffset(2, "Error getting event-display Knative Service: %s", err)
	}

	return ksvcDeploymentName(c, ksvc)
}

// ksvcDeployment returns the name of the Deployment matching the latest
// revision of the given Knative Service.
func ksvcDeploymentName(c dynamic.Interface, ksvc *unstructured.Unstructured) string {
	latestRev, found, err := unstructured.NestedString(ksvc.Object, "status", "latestCreatedRevisionName")
	if err != nil {
		framework.FailfWithOffset(3, "Error reading status.latestCreatedRevisionName field: %s", err)
	}
	if !found {
		framework.FailfWithOffset(3, "The Knative Service did not report its latestCreatedRevisionName")
	}

	var rev kmeta.Accessor = &unstructured.Unstructured{}
	rev.SetName(latestRev)

	return revisionnames.Deployment(rev)
}

// ReceivedEventDisplayEvents returns all events found in the given
// event-display log stream.
func ReceivedEventDisplayEvents(logStream io.ReadCloser) []cloudevents.Event {
	defer func() {
		if err := logStream.Close(); err != nil {
			framework.FailfWithOffset(3, "Failed to close event-display's log stream: %s", err)
		}
	}()

	eventStrings := splitEvents(logStream)

	if len(eventStrings) == 0 {
		return nil
	}

	events := make([]cloudevents.Event, len(eventStrings))

	for i, eStr := range eventStrings {
		events[i] = parseCloudEvent(eStr)
	}

	return events
}

// splitEvents parses the given stream into a list of event strings.
func splitEvents(logStream io.Reader) []string {
	const delimiter = '☁'

	var buf bytes.Buffer

	// read everything at once instead of per chunk, because a buffered
	// read could end in the middle of a delimiter rune, causing an entire
	// event to be overlooked while parsing
	if _, err := buf.ReadFrom(logStream); err != nil {
		framework.FailfWithOffset(2, "Error reading event-display's log stream: %s", err)
	}

	eventBuilders := make([]strings.Builder, 0)
	currentEventIdx := -1

	for {
		r, _, err := buf.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			framework.FailfWithOffset(2, "Error reading buffer: %s", err)
		}

		if r == delimiter {
			eventBuilders = append(eventBuilders, make([]strings.Builder, 1)...)
			currentEventIdx++
		}
		// everything until the first found delimiter will be ignored
		if currentEventIdx >= 0 {
			eventBuilders[currentEventIdx].WriteRune(r)
		}
	}

	events := make([]string, len(eventBuilders))

	for i, eb := range eventBuilders {
		events[i] = eb.String()
	}

	return events
}

// parseCloudEvent parses the content of a stringified CloudEvent into a
// structured CloudEvent.
//
// Example of output from Event.String():
//
// ☁  cloudevents.Event
// Validation: valid
// Context Attributes,
//   specversion: 1.0
//   type: io.triggermesh.some.event
//   source: some/source
//   subject: some-subject
//   id: edecf007-f651-4e10-959e-e2f0a5b8ccd0
//   time: 2020-09-14T13:59:40.693213706Z
//   datacontenttype: application/json
// Extensions,
//   someextension: some-value
// Data,
//   {
//     ...
//   }
func parseCloudEvent(ce string) cloudevents.Event {
	e := cloudevents.NewEvent()

	contentType := cloudevents.ApplicationJSON

	r := bufio.NewReader(strings.NewReader(ce))

	for {
		line, err := r.ReadString('\n')
		line = strings.TrimSpace(line)

		// try finding context attributes and data
		subs := strings.SplitN(line, ":", 2)

		switch len(subs) {
		case 1:
			if subs[0] != "Data," {
				break
			}

			// read everything that's left to read and set
			// it as the event's data
			b, err := ioutil.ReadAll(r)
			if err != nil {
				framework.Logf("Error reading event's data: %s", err)
				break
			}

			decodedData := make(map[string]interface{})

			if err := datacodec.Decode(context.Background(), e.DataMediaType(), b, &decodedData); err != nil {
				framework.Logf("Error decoding event's data, using raw bytes instead: %s", err)
				if err := e.SetData(contentType, b); err != nil {
					framework.Logf("Error setting event's data: %s", err)
				}
			} else {
				if err := e.SetData(contentType, decodedData); err != nil {
					framework.Logf("Error setting event's data: %s", err)
				}
			}

		case 2:
			switch k, v := subs[0], strings.TrimSpace(subs[1]); k {

			// Required attributes
			case "id":
				e.SetID(v)
			case "source":
				e.SetSource(v)
			case "specversion":
				e.SetSpecVersion(v)
			case "type":
				e.SetType(v)

			// Optional attributes
			case "datacontenttype":
				contentType = v
				e.SetDataContentType(v)
			case "dataschema":
				e.SetDataSchema(v)
			case "subject":
				e.SetSubject(v)
			case "time":
				t, err := time.Parse(time.RFC3339Nano, v)
				if err != nil {
					framework.Logf("Error parsing event's time: %s", err)
					break
				}
				e.SetTime(t)

			// Extensions
			default:
				e.SetExtension(k, v)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			framework.FailfWithOffset(3, "Error reading line from Reader: %s", err)
		}
	}

	return e
}
