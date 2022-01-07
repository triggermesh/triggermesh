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

package awssnssource

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

// ensureSubscribed ensures the source's HTTP(S) endpoint is subscribed to the
// SNS topic.
func (r *Reconciler) ensureSubscribed(ctx context.Context) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.AWSSNSSource)
	status := &src.Status

	isDeployed := status.GetCondition(v1alpha1.ConditionDeployed).IsTrue()

	// skip this cycle if the URL couldn't yet be determined
	if !isDeployed {
		status.MarkNotSubscribed(v1alpha1.AWSSNSReasonNoURL, "The receive adapter isn't ready yet")
		return nil
	}

	snsClient, err := r.snsCg.Get(src)
	if err != nil {
		status.MarkNotSubscribed(v1alpha1.AWSSNSReasonNoClient, "Cannot obtain SNS client")
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error creating SNS client: %s", err))
	}

	url := status.Address.URL
	topicARN := src.Spec.ARN.String()

	// We need to check the existence of the topic prior to attempting to
	// list its subscriptions, because AWS returns the same NotFound error
	// whether the topic or the subscription is missing, and we want to
	// handle both cases differently.
	topicExists, err := checkTopicExists(ctx, snsClient, topicARN)
	switch {
	case isDenied(err):
		msg := fmt.Sprintf("Authorization error checking existence of topic %q: %s", topicARN, toErrMsg(err))
		status.MarkNotSubscribed(v1alpha1.AWSSNSReasonAPIError, msg)
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe, msg))
	case err != nil:
		msg := fmt.Sprintf("Error checking existence of topic %q: %s", topicARN, toErrMsg(err))
		status.MarkNotSubscribed(v1alpha1.AWSSNSReasonAPIError, msg)
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe, msg))
	case !topicExists:
		msg := fmt.Sprintf("The provided topic %q does not exist", topicARN)
		status.MarkNotSubscribed(v1alpha1.AWSSNSReasonFailedSync, msg)
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe, msg))
	}

	subsARN, err := findSubscription(ctx, snsClient, topicARN, url.String())
	switch {
	case isPending(subsARN), isNotFound(err):
		subsARN, err = subscribe(ctx, snsClient, topicARN, url, src.Spec.SubscriptionAttributes)
		switch {
		case isPending(subsARN):
			status.MarkNotSubscribed(v1alpha1.AWSSNSReasonPending, "Subscription is pending confirmation")
			// wrap to fail the reconciliation
			return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
				"Subscription is pending confirmation, will retry"))
		case isAWSError(err):
			status.MarkNotSubscribed(v1alpha1.AWSSNSReasonRejected, "Subscription request rejected")
			event := subscribeErrorEvent(url, topicARN, err)

			// Most documented API errors require some user intervention and
			// are not to be retried.
			// https://docs.aws.amazon.com/sns/latest/api/API_Subscribe.html#API_Subscribe_Errors
			//
			// However "InvalidParameter" is returned if the subscription URL is not ready, which
			// might be the case if the endpoint for the source is not exposed yet.
			if awsErr := awserr.Error(nil); errors.As(err, &awsErr) && awsErr.Code() == sns.ErrCodeInvalidParameterException {
				return event
			}

			return controller.NewPermanentError(event)
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.AWSSNSReasonFailedSync, "Cannot subscribe endpoint")
			return fmt.Errorf("%w", subscribeErrorEvent(url, topicARN, err))
		}

	case isAWSError(err):
		// All documented API errors require some user intervention and
		// are not to be retried.
		// https://docs.aws.amazon.com/sns/latest/api/API_Subscribe.html#API_Subscribe_Errors
		status.MarkNotSubscribed(v1alpha1.AWSSNSReasonAPIError, "API call rejected")
		return controller.NewPermanentError(apiErrorEvent(err))

	case err != nil:
		return fmt.Errorf("finding subscription: %w", err)
	}

	if !status.GetCondition(v1alpha1.AWSSNSConditionSubscribed).IsTrue() {
		event.Normal(ctx, ReasonSubscribed, "Subscribed to SNS topic %q", topicARN)
	}
	status.MarkSubscribed(subsARN)

	return nil
}

// ensureUnsubscribed ensures the source's HTTP(S) endpoint is unsubscribed
// from the SNS topic.
func (r *Reconciler) ensureUnsubscribed(ctx context.Context) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.AWSSNSSource)
	status := src.Status

	snsClient, err := r.snsCg.Get(src)
	switch {
	case isNotFound(err):
		// the finalizer is unlikely to recover from a missing Secret,
		// so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Secret missing while finalizing event source. Ignoring: %s", err)
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error creating SNS client: %s", err)
	}

	topicARN := src.Spec.ARN.String()

	// We need to check the existence of the topic prior to attempting to
	// list its subscriptions, because AWS returns the same NotFound error
	// whether the topic or the subscription is missing, and we want to
	// handle both cases differently.
	topicExists, err := checkTopicExists(ctx, snsClient, topicARN)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error checking the existence of the topic. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error checking the existence of topic: %s", toErrMsg(err))
	}

	var subsARN string
	if sa := src.Status.SubscriptionARN; sa != nil {
		subsARN = *sa
	}

	// Prefer getting the subscription's ARN from AWS directly if the topic
	// exists, as it is more reliable than what may be stored in the source's status.
	if topicExists && status.Address != nil && status.Address.URL != nil {
		subsARN, err = findSubscription(ctx, snsClient, topicARN, status.Address.URL.String())
		switch {
		case isPending(subsARN):
			return reconciler.NewEvent(corev1.EventTypeNormal, ReasonFailedUnsubscribe,
				"Subscription wasn't confirmed, skipping finalization")
		case isNotFound(err):
			return reconciler.NewEvent(corev1.EventTypeNormal, ReasonUnsubscribed,
				"Subscription already absent, skipping finalization")
		case isDenied(err):
			// it is unlikely that we recover from auth errors in the
			// finalizer, so we simply record a warning event and return
			event.Warn(ctx, ReasonFailedUnsubscribe,
				"Authorization error finding subscription. Ignoring: %s", toErrMsg(err))
			return nil
		case err != nil:
			return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
				"Error finding subscription: %s", toErrMsg(err))
		}
	}

	// It is possible that the source never successfully subscribed to the
	// topic at all, in which case there is nothing to unsubscribe.
	if subsARN == "" {
		return nil
	}

	err = unsubscribe(ctx, snsClient, subsARN)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error unsubscribing from SNS topic %q. Ignoring: %s", topicARN, toErrMsg(err))
		return nil
	case err != nil:
		return unsubscribeErrorEvent(topicARN, err)
	}

	return reconciler.NewEvent(corev1.EventTypeNormal, ReasonUnsubscribed,
		"Unsubscribed from SNS topic %q", topicARN)
}

// findSubscription returns the ARN of the subscription corresponding to the
// given topic URL if it exists.
func findSubscription(ctx context.Context, cli snsiface.SNSAPI, topicARN, endpointURL string) (string /*arn*/, error) {
	in := &sns.ListSubscriptionsByTopicInput{
		TopicArn: &topicARN,
	}

	out := &sns.ListSubscriptionsByTopicOutput{}

	var err error

	initialRequest := true

	for out.NextToken != nil || initialRequest {
		in.NextToken = out.NextToken

		out, err = cli.ListSubscriptionsByTopicWithContext(ctx, in)
		if err != nil {
			return "", fmt.Errorf("listing subscriptions for topic %q: %w", topicARN, err)
		}

		if initialRequest {
			initialRequest = false
		}

		for _, sub := range out.Subscriptions {
			if *sub.Endpoint == endpointURL {
				return *sub.SubscriptionArn, nil
			}
		}
	}

	return "", awserr.New(sns.ErrCodeNotFoundException, "", nil)
}

// checkTopicExists returns whether the topic with the given ARN exists.
func checkTopicExists(ctx context.Context, cli snsiface.SNSAPI, topicARN string) (bool, error) {
	in := &sns.GetTopicAttributesInput{
		TopicArn: &topicARN,
	}

	_, err := cli.GetTopicAttributesWithContext(ctx, in)
	switch {
	case isNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("getting topic attributes: %w", err)
	}

	return true, nil
}

// subscribe subscribes to a SNS topic.
func subscribe(ctx context.Context, cli snsiface.SNSAPI, topicARN string,
	endpointURL *apis.URL, attributes map[string]*string) (string /*arn*/, error) {

	resp, err := cli.SubscribeWithContext(ctx, &sns.SubscribeInput{
		Endpoint:   aws.String(endpointURL.String()),
		Protocol:   &endpointURL.Scheme,
		TopicArn:   &topicARN,
		Attributes: attributes,
	})
	if err != nil {
		return "", fmt.Errorf("subscribing to topic: %w", err)
	}

	logging.FromContext(ctx).Debug("Subscribe responded with: ", resp)

	return *resp.SubscriptionArn, nil
}

// unsubscribe unsubscribes from a SNS topic.
func unsubscribe(ctx context.Context, cli snsiface.SNSAPI, subsARN string) error {
	// NOTE(antoineco): this API call does not seem to return any error if
	// the given SubscriptionArn can't be found
	resp, err := cli.UnsubscribeWithContext(ctx, &sns.UnsubscribeInput{
		SubscriptionArn: &subsARN,
	})
	if err != nil {
		return fmt.Errorf("unsubscribing from topic: %w", err)
	}

	logging.FromContext(ctx).Debug("Unsubscribe responded with: ", resp)

	return nil
}

// isNotFound returns whether the given error indicates that some resource was
// not found.
func isNotFound(err error) bool {
	if k8sErr := apierrors.APIStatus(nil); errors.As(err, &k8sErr) {
		return k8sErr.Status().Reason == metav1.StatusReasonNotFound
	}
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		return awsErr.Code() == sns.ErrCodeNotFoundException
	}
	return false
}

// isDenied returns whether the given error indicates that a request to the SNS
// API could not be authorized.
func isDenied(err error) bool {
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		if awsReqFail := awserr.RequestFailure(nil); errors.As(err, &awsReqFail) {
			code := awsReqFail.StatusCode()
			return code == http.StatusUnauthorized || code == http.StatusForbidden
		}

		return awsErr.Code() == sns.ErrCodeAuthorizationErrorException
	}
	return false
}

// isAWSError returns whether the given error is an AWS API error.
func isAWSError(err error) bool {
	awsErr := awserr.Error(nil)
	return errors.As(err, &awsErr)
}

// isPending returns whether the given ARN string indicates that the
// subscription is still pending.
func isPending(arn string) bool {
	const (
		// returned by ListSubscriptionsByTopic
		listSubsPending = "PendingConfirmation"
		// returned by Subscribe
		subscribePending = "pending confirmation"
	)

	return arn == listSubsPending || arn == subscribePending
}

// toErrMsg attempts to extract the message from the given error if it is an
// AWS error.
// Those errors are particularly verbose and include a unique request ID that
// causes an infinite loop of reconciliations when appended to a status
// condition. Some AWS errors are not recoverable without manual intervention
// (e.g. invalid secrets) so there is no point letting that behaviour happen.
func toErrMsg(err error) string {
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		return awserr.SprintError(awsErr.Code(), awsErr.Message(), "", awsErr.OrigErr())
	}
	return err.Error()
}

// subscribeErrorEvent returns a reconciler event indicating that an endpoint
// could not be subscribed to a SNS topic.
func subscribeErrorEvent(url *apis.URL, topicARN string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error subscribing endpoint %q to SNS topic %q: %s", url, topicARN, toErrMsg(origErr))
}

// apiErrorEvent returns a reconciler event indicating that the SNS API
// returned an error.
func apiErrorEvent(origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error communicating with the SNS API: %s", toErrMsg(origErr))
}

// unsubscribeErrorEvent returns a reconciler event indicating that an endpoint
// could not be unsubscribed from a SNS topic.
func unsubscribeErrorEvent(topicARN string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
		"Error unsubscribing from SNS topic %q: %s", topicARN, toErrMsg(origErr))
}
