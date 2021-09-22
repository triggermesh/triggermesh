/*
Copyright (c) 2021 TriggerMesh Inc.

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

package event

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"knative.dev/pkg/controller"
)

func TestEvents(t *testing.T) {
	const (
		reason     = "Testing"
		messageFmt = `message: %s`
		arg        = "hello world"
	)

	const eventRecorderBufferSize = 10
	er := record.NewFakeRecorder(eventRecorderBufferSize)

	ctx := controller.WithEventRecorder(context.TODO(), er)

	Record(ctx, &corev1.Pod{}, corev1.EventTypeNormal, reason, messageFmt, arg)
	Record(ctx, &corev1.Service{}, corev1.EventTypeWarning, reason, messageFmt, arg)

	close(er.Events)

	const expectEvents = 2

	recordedEvent := make([]string, 0, expectEvents)
	for ev := range er.Events {
		recordedEvent = append(recordedEvent, ev)
	}
	assert.Len(t, recordedEvent, expectEvents, "Expect %d events", expectEvents)

	expectNormalEventContent := eventf(corev1.EventTypeNormal, reason, messageFmt, arg)
	expectWarningEventContent := eventf(corev1.EventTypeWarning, reason, messageFmt, arg)

	assert.Equalf(t, expectNormalEventContent, recordedEvent[0], "Expect message content to match input")
	assert.Equalf(t, expectWarningEventContent, recordedEvent[1], "Expect message content to match input")
}

// eventf returns the attributes of an API event in the format returned by
// Kubernetes' FakeRecorder.
func eventf(eventtype, reason, messageFmt string, args ...interface{}) string {
	return fmt.Sprintf(eventtype+" "+reason+" "+messageFmt, args...)
}
