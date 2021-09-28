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

package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"knative.dev/pkg/controller"

	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	eventtesting "github.com/triggermesh/triggermesh/pkg/routing/testing/event"
)

func TestEvents(t *testing.T) {
	const (
		reason     = "Testing"
		messageFmt = `message: %s`
		normalMsg  = "normal event"
		warningMsg = "warning event"
	)

	const eventRecorderBufferSize = 10
	er := record.NewFakeRecorder(eventRecorderBufferSize)

	ctx := controller.WithEventRecorder(context.TODO(), er)
	ctx = v1alpha1.WithRouter(ctx, &v1alpha1.Filter{})

	Normal(ctx, reason, messageFmt, normalMsg)
	Warn(ctx, reason, messageFmt, warningMsg)

	close(er.Events)

	const expectEvents = 2

	recordedEvent := make([]string, 0, expectEvents)
	for ev := range er.Events {
		recordedEvent = append(recordedEvent, ev)
	}
	require.Len(t, recordedEvent, expectEvents, "Expect %d events", expectEvents)

	expectNormalEventContent := eventtesting.Eventf(corev1.EventTypeNormal, reason, messageFmt, normalMsg)
	expectWarningEventContent := eventtesting.Eventf(corev1.EventTypeWarning, reason, messageFmt, warningMsg)

	assert.Equalf(t, expectNormalEventContent, recordedEvent[0], "Expect message content to match input")
	assert.Equalf(t, expectWarningEventContent, recordedEvent[1], "Expect message content to match input")
}
