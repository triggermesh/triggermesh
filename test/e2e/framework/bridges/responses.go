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

package bridges

import (
	"context"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	"knative.dev/eventing/pkg/apis/messaging"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

const replyToNamePrefix = "replyto-"

const (
	k8sAPIVersion = "apiVersion"
	k8sKind       = "kind"
	k8sMetaName   = "name"
)

// SetupSubscriberWithReplyTo sets up the infrastructure required for an
// Addressable to receive events and emit replies which are automatically
// routed to an event-display event sink.
// It does it by fronting the 'dst' Addressable with a Channel, and configuring
// a Subscription that routes replies to an instance of event-display.
// The function returns:
// - the URL of the Channel which events can be sent to
// - the name of the event-display sink Deployment which replies are sent to
func SetupSubscriberWithReplyTo(cli clientset.Interface, dynCli dynamic.Interface,
	namespace string, dst *duckv1.Destination) (entrypoint *url.URL, replyDstDeplName string) {

	ctx := context.Background()

	channelGVR := messaging.ChannelsResource.WithVersion("v1")
	channelAPIVersion := channelGVR.GroupVersion().String()
	channelKind := (*messagingv1.Channel)(nil).GetGroupVersionKind().Kind

	ch := &unstructured.Unstructured{}
	ch.SetAPIVersion(channelAPIVersion)
	ch.SetKind(channelKind)
	ch.SetGenerateName(replyToNamePrefix)

	ch, err := dynCli.Resource(channelGVR).Namespace(namespace).Create(ctx, ch, metav1.CreateOptions{})
	if err != nil {
		framework.FailfWithOffset(1, "Error creating Channel: %s", err)
	}

	repliesDisplay := CreateEventDisplaySink(cli, namespace)

	subsGVR := messaging.SubscriptionsResource.WithVersion("v1")
	subsAPIVersion := subsGVR.GroupVersion().String()
	subsKind := (*messagingv1.Subscription)(nil).GetGroupVersionKind().Kind

	subs := &unstructured.Unstructured{}
	subs.SetAPIVersion(subsAPIVersion)
	subs.SetKind(subsKind)
	subs.SetName(ch.GetName())

	subsCh := map[string]interface{}{
		k8sAPIVersion: channelAPIVersion,
		k8sKind:       channelKind,
		k8sMetaName:   ch.GetName(),
	}

	subsSink := map[string]interface{}{
		k8sAPIVersion: dst.Ref.APIVersion,
		k8sKind:       dst.Ref.Kind,
		k8sMetaName:   dst.Ref.Name,
	}

	subsReplyTo := map[string]interface{}{
		k8sAPIVersion: repliesDisplay.Ref.APIVersion,
		k8sKind:       repliesDisplay.Ref.Kind,
		k8sMetaName:   repliesDisplay.Ref.Name,
	}

	_ = unstructured.SetNestedMap(subs.Object, subsCh, "spec", "channel")
	_ = unstructured.SetNestedMap(subs.Object, subsSink, "spec", "subscriber", "ref")
	_ = unstructured.SetNestedMap(subs.Object, subsReplyTo, "spec", "reply", "ref")

	if _, err = dynCli.Resource(subsGVR).Namespace(namespace).Create(ctx, subs, metav1.CreateOptions{}); err != nil {
		framework.FailfWithOffset(1, "Error creating Subscription: %s", err)
	}

	ch = ducktypes.WaitUntilReady(dynCli, ch)

	return ducktypes.Address(ch), repliesDisplay.Ref.Name
}
