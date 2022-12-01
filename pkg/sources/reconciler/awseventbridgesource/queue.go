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

package awseventbridgesource

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"reflect"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/aws/aws-sdk-go/aws/arn"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/triggermesh/triggermesh/pkg/apis"
	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/aws/iam"
	"github.com/triggermesh/triggermesh/pkg/sources/aws/sqs"
)

// SQSQueue wraps information about a SQS queue.
type SQSQueue struct {
	URL    string
	ARN    string
	Policy string
}

// EnsureQueue ensures the existence of a SQS queue for sending EventBridge events.
func EnsureQueue(ctx context.Context, cli sqsiface.SQSAPI) (*SQSQueue, error) {
	if skip.Skip(ctx) {
		return &SQSQueue{}, nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	status := &typedSrc.Status

	if dest := typedSrc.Spec.Destination; dest != nil {
		if userProvidedQueue := dest.SQS; userProvidedQueue != nil {
			status.QueueARN = &userProvidedQueue.QueueARN
			return &SQSQueue{
				ARN: userProvidedQueue.QueueARN.String(),
			}, nil
		}
	}

	queueName := queueName(typedSrc)

	queueURL, err := sqs.QueueURL(cli, queueName)
	switch {
	case isNotFound(err):
		queueURL, err = sqs.CreateQueue(cli, queueName, queueTags(typedSrc))
		if err != nil {
			status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Unable to create SQS queue")
			return nil, fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedQueue,
				"Error creating SQS queue for EventBridge events: %s", toErrMsg(err)))
		}
		event.Normal(ctx, ReasonQueueCreated, "Created SQS queue %q", queueURL)

	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Request to SQS API got rejected")
		return nil, controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Authorization error interacting with the SQS API: %s", toErrMsg(err)))

	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Cannot synchronize SQS queue")
		return nil, fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Failed to determine URL of SQS queue: %s", toErrMsg(err)))
	}

	getAttrs := []string{awssqs.QueueAttributeNameQueueArn, awssqs.QueueAttributeNamePolicy}
	queueAttrs, err := sqs.QueueAttributes(cli, queueURL, getAttrs)
	if err != nil {
		return nil, fmt.Errorf("getting attributes of SQS queue: %w", err)
	}

	queueARN := queueAttrs[awssqs.QueueAttributeNameQueueArn]

	queueARNStruct, err := arnStrToARN(queueARN)
	if err != nil {
		return nil, fmt.Errorf("converting ARN string to structured ARN: %w", err)
	}

	// it is essential that we propagate the queue's ARN here,
	// otherwise BuildAdapter() won't be able to configure the SQS
	// adapter properly
	status.QueueARN = queueARNStruct

	queuePolicy := queueAttrs[awssqs.QueueAttributeNamePolicy]

	return &SQSQueue{
		URL:    queueURL,
		ARN:    queueARN,
		Policy: queuePolicy,
	}, nil
}

// EnsureNoQueue ensures that the SQS queue used for sending EventBridge events is deleted.
func EnsureNoQueue(ctx context.Context, cli sqsiface.SQSAPI, queueName string) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	if dest := typedSrc.Spec.Destination; dest != nil {
		if userProvidedQueue := dest.SQS; userProvidedQueue != nil {
			// do not delete queues managed by the user
			return nil
		}
	}

	queueURL, err := sqs.QueueURL(cli, queueName)
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, "Queue not found, skipping deletion")
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error getting SQS queue. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to determine URL of SQS queue: %s", toErrMsg(err))
	}

	owns, err := assertQueueOwnership(cli, queueURL, typedSrc)
	if err != nil {
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to verify owner of SQS queue: %s", toErrMsg(err))
	}

	if !owns {
		event.Warn(ctx, ReasonUnsubscribed, "Queue %q is not owned by this source instance, "+
			"skipping deletion", queueURL)
		return nil
	}

	err = sqs.DeleteQueue(cli, queueURL)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error deleting SQS queue. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error deleting SQS queue: %s", toErrMsg(err))
	}

	event.Normal(ctx, ReasonQueueDeleted, "Deleted SQS queue %q", queueURL)

	return nil
}

// EnsureQueuePolicy ensures that the correct access policy is applied to the
// given SQS queue.
func EnsureQueuePolicy(ctx context.Context, cli sqsiface.SQSAPI, queue *SQSQueue, ruleARN *apis.ARN) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSEventBridgeSource)

	status := &typedSrc.Status

	if dest := typedSrc.Spec.Destination; dest != nil {
		if userProvidedQueue := dest.SQS; userProvidedQueue != nil {
			return nil
		}
	}

	currentPol := unmarshalQueuePolicy(queue.Policy)
	desiredPol := makeQueuePolicy(queue.ARN, ruleARN.String(), typedSrc)

	err := syncQueuePolicy(cli, queue.URL, currentPol, desiredPol)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Request to SQS API got rejected")
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Authorization error setting access policy: %s", toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSEventBridgeReasonAPIError, "Cannot synchronize SQS queue")
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error synchronizing policy of SQS queue: %s", toErrMsg(err)))
	}

	return nil
}

// syncQueuePolicy ensures that a SQS queue has the right permissions to
// receive messages from the EventBridge event bus observed by the given source.
func syncQueuePolicy(cli sqsiface.SQSAPI, queueURL string, current, desired iam.Policy) error {
	if equalPolicies(desired, current) {
		return nil
	}

	if err := sqs.SetQueuePolicy(cli, queueURL, desired); err != nil {
		return fmt.Errorf("setting policy of SQS queue: %w", err)
	}

	return nil
}

// equalPolicies returns whether two SQS policies are semantically equal.
func equalPolicies(a, b iam.Policy) bool {
	if len(a.Statement) != len(b.Statement) {
		return false
	}

	as, bs := a.Statement[0], b.Statement[0]

	if !reflect.DeepEqual(as.Principal, bs.Principal) {
		return false
	}
	if !reflect.DeepEqual(as.Condition, bs.Condition) {
		return false
	}
	if !reflect.DeepEqual(as.Action, bs.Action) {
		return false
	}
	return reflect.DeepEqual(as.Resource, bs.Resource)
}

// makeQueuePolicy creates an IAM policy for the given SQS queue ARN and source instance.
func makeQueuePolicy(queueARN, ruleARN string, src *v1alpha1.AWSEventBridgeSource) iam.Policy {
	accID := src.Spec.ARN.AccountID

	return iam.NewPolicy(
		newEventBridgeToSQSPolicyStatement(queueARN, ruleARN, accID),
	)
}

// newEventBridgeToSQSPolicyStatement returns an IAM Policy Statement that
// allows an EventBridge event bus to publish events to the given SQS queue.
// Ref. https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-targets.html#targets-permissions
func newEventBridgeToSQSPolicyStatement(queueARN, ruleARN, accID string) iam.PolicyStatement {
	return iam.NewPolicyStatement(iam.EffectAllow,
		iam.PrincipalService("events.amazonaws.com"),
		iam.ConditionArnEquals("aws:SourceArn", ruleARN),
		iam.ConditionStringEquals("aws:SourceAccount", accID),
		iam.Action("sqs:SendMessage"),
		iam.Resource(queueARN),
	)
}

// unmarshalQueuePolicy deserializes an IAM policy string.
func unmarshalQueuePolicy(polStr string) iam.Policy {
	var pol iam.Policy
	_ = json.Unmarshal([]byte(polStr), &pol)

	// if an error occured, the policy will be empty syncQueuePolicy() will
	// simply enforce the desired state
	return pol
}

// queueName returns a deterministic name for the reconciled SQS queue.
//
// A queue name can have up to 80 characters, and contain only alphanumeric
// characters, hyphens (-), and underscores (_), which doesn't give us a lot of
// characters for indicating what component owns the queue. Therefore, we
// compute the CRC32 checksum of the source's name/namespace (8 characters) and
// make it part of the name.
func queueName(src *v1alpha1.AWSEventBridgeSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.Namespace + "/" + src.Name))
	return "io_triggermesh_awseventbridgesources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// assertQueueOwnership returns whether a SQS queue identified by URL is owned
// by the given source.
func assertQueueOwnership(cli sqsiface.SQSAPI, queueURL string, src *v1alpha1.AWSEventBridgeSource) (bool, error) {
	tags, err := sqs.QueueTags(cli, queueURL)
	if err != nil {
		return false, fmt.Errorf("listing tags of SQS queue: %w", err)
	}

	return tags["owned-by"] == sourceID(src), nil
}

// queueTags returns a set of tags containing information from the given source
// instance to set on a SQS queue.
func queueTags(src *v1alpha1.AWSEventBridgeSource) map[string]string {
	return map[string]string{
		"eventbus-arn": src.Spec.ARN.String(),
		"owned-by":     sourceID(src),
	}
}

// arnStrToARN returns the given ARN string as a structured ARN.
func arnStrToARN(arnStr string) (*apis.ARN, error) {
	arn, err := arn.Parse(arnStr)
	if err != nil {
		return nil, fmt.Errorf("parsing ARN string: %w", err)
	}

	apiARN := apis.ARN(arn)
	return &apiARN, nil
}
