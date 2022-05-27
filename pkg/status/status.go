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

// Package status contains helpers to observe the status of Kubernetes objects.
package status

import (
	"strings"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	corelistersv1 "k8s.io/client-go/listers/core/v1"

	"knative.dev/pkg/apis"
)

const (
	reasonAppRuntimeFailure = "AppRuntimeFailure"
	reasonBadContainerImage = "BadContainerImage"

	reasonMissingPrefix = "Missing"

	// https://github.com/knative/serving/blob/release-0.16/pkg/apis/serving/v1/configuration_lifecycle.go#L91
	knRevisionFailedReason = "RevisionFailed"
)

// DeploymentPodsWaitingState collects the Pods owned by the given Deployment
// and returns the state of the first observed Pod in a "waiting" state, or nil
// if no Pod is found in that state.
func DeploymentPodsWaitingState(d *appsv1.Deployment,
	pl corelistersv1.PodNamespaceLister) (*corev1.ContainerStateWaiting, error) {

	sel, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, err
	}

	pods, err := pl.List(sel)
	if err != nil {
		return nil, err
	}

	for _, p := range pods {
		if p.Status.Phase == corev1.PodRunning || p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, ps := range p.Status.ContainerStatuses {
			if ws := ps.State.Waiting; ws != nil {
				return ws, nil
			}
		}
	}

	return nil, nil
}

// ExactReason tries to determine the exact reason of a failure from a
// container state or Knative status condition. The format of the returned
// reason follows the CamelCased one-word convention described at
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
// When a more specific reason can not be determined, the original reason is
// returned.
func ExactReason(state interface{}) string /*reason*/ {
	switch in := state.(type) {
	case *corev1.ContainerStateWaiting:
		return exactReason(in.Reason, in.Message)
	case *apis.Condition:
		return exactReason(in.Reason, in.Message)
	}
	return ""
}

func exactReason(reason, msg string) string /*reason*/ {
	if isRuntimeError(reason, msg) {
		return reasonAppRuntimeFailure
	}

	if isBadImageError(reason, msg) {
		return reasonBadContainerImage
	}

	if missing, typ := isResourceMissingError(reason, msg); missing {
		return reasonMissingPrefix + typ
	}

	return reason
}

// isRuntimeError returns whether the given combination of reason and message
// indicates that a container within a Pod is failing to run.
func isRuntimeError(reason, msg string) bool {
	// https://github.com/knative/serving/blob/release-0.16/pkg/apis/serving/v1/revision_lifecycle.go#L203
	const knContainerFailurePrefix = "Container failed with: "

	return runtimeErrorReasons.Has(reason) ||
		reason == knRevisionFailedReason && strings.Contains(msg, knContainerFailurePrefix)
}

// isBadImageError returns whether the given combination of reason and message
// indicates that a container image can not be pulled.
func isBadImageError(reason, msg string) bool {
	// https://github.com/knative/serving/blob/release-0.16/pkg/apis/serving/v1/revision_lifecycle.go#L209
	const knFetchImgErrorPrefix = `Unable to fetch image "`

	return imagePullErrorReasons.Has(reason) ||
		// Knative Serving uses its own image resolver to determine whether an
		// image can be pulled, therefore the message indicating a failure can
		// be considered stable.
		// https://github.com/knative/serving/blob/release-0.16/pkg/reconciler/revision/revision.go#L122
		reason == knRevisionFailedReason && strings.Contains(msg, knFetchImgErrorPrefix)
}

// isResourceMissingError returns whether the given combination of reason and
// message indicates that a resource is missing. If that's the case, the type
// of the resource is also returned.
// This parsing logic is tailored to the error message to avoid the use of
// regular expressions.
// The expected format comes from the standard Kubernetes API errors:
// https://github.com/kubernetes/kubernetes/blob/release-1.18/staging/src/k8s.io/apimachinery/pkg/api/errors/errors.go#L139
func isResourceMissingError(reason, msg string) (bool, string /*resource type*/) {
	const quoteNotFound = `" not found`

	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
	const maxK8sNameLength = 253

	if !resourceMissingErrorReasons.Has(reason) {
		return false, ""
	}

	msg = strings.TrimSuffix(msg, ".")
	if !strings.HasSuffix(msg, quoteNotFound) {
		return false, ""
	}

	// We know the message is likely to have the expected format based on
	// its suffix, so we try to extract the resource type:
	//
	//   `failed to do xyz: secret "my-api-token" not found`
	//                      ^^^^^^
	//
	// To do so, we first ensure the message includes an opening quote:
	//
	//   `failed to do xyz: secret "my-api-token" not found`
	//                             ^
	//
	// If that is the case, we parse the string in reverse from the
	// position of that quote until the beginning of a word:
	//
	//   `failed to do xyz: secret "my-api-token" not found`
	//                      |<-----^

	// index of the closing and opening quote chars
	closingResNameQuoteIdx := len(msg) - len(quoteNotFound)
	openingResNameQuoteIdx := strings.LastIndex(msg[:closingResNameQuoteIdx], `"`)

	// we couldn't find the opening quote or it's at the beginning of the message
	if openingResNameQuoteIdx <= 0 ||
		// the quoted string exceeds the maximum length of a Kubernetes object
		(closingResNameQuoteIdx-openingResNameQuoteIdx)+1 > maxK8sNameLength ||
		// the preceding char is not a space
		msg[openingResNameQuoteIdx-1] != ' ' {

		return false, ""
	}

	// parse the type in reverse starting from before the opening quote
	typeBeginningIdx := openingResNameQuoteIdx - 1
	for i := openingResNameQuoteIdx - 1; i > 0; i-- {
		if unicode.IsLetter(rune(msg[i-1])) {
			// if we reached the beginning of the message, it means
			// the type is at the beginning of the message
			if i-1 == 0 {
				typeBeginningIdx = 0
			}
			continue
		}
		// break on the first occurrence of a non-letter char
		typeBeginningIdx = i
		break
	}
	// edge case: the resource name is empty
	if (openingResNameQuoteIdx-1)-typeBeginningIdx == 0 {
		return false, ""
	}

	var typ strings.Builder
	for i, char := range msg[typeBeginningIdx:(openingResNameQuoteIdx - 1)] {
		if i == 0 {
			typ.WriteRune(unicode.ToUpper(char))
			continue
		}
		typ.WriteRune(char)
	}

	return true, typ.String()
}

// runtimeErrorReasons is a set of status reasons that could indicate that a
// container is failing to run.
// https://github.com/kubernetes/kubernetes/blob/release-1.17/pkg/kubelet/container/sync_result.go#L29-L45
var runtimeErrorReasons = sets.NewString(
	"CrashLoopBackOff",
	"RunContainerError",
	"RunInitContainerError",
	"CreatePodSandboxError",
	"ConfigPodSandboxError",
	"VerifyNonRootError",
)

// imagePullErrorReasons is a set of status reasons that could indicate that a
// container image can not be pulled.
// https://github.com/kubernetes/kubernetes/blob/release-1.17/pkg/kubelet/images/types.go#L26-L44
var imagePullErrorReasons = sets.NewString(
	"ImagePullBackOff",
	"ImageInspectError",
	"ErrImagePull",
	"ErrImageNeverPull",
	"RegistryUnavailable",
	"InvalidImageName",
)

// resourceMissingErrorReasons is a set of status reasons that could indicate that a
// container is missing required resources.
// https://github.com/kubernetes/kubernetes/blob/release-1.17/pkg/kubelet/kuberuntime/kuberuntime_container.go#L55-L58
var resourceMissingErrorReasons = sets.NewString(
	"CreateContainerConfigError",
	"CreateContainerError",
	knRevisionFailedReason,
)
