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

package awss3source

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sqs"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
)

// EnsureNotificationsEnabled ensures that event notifications are enabled in
// the S3 bucket.
func EnsureNotificationsEnabled(ctx context.Context, cli s3iface.S3API, queueARN string) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSS3Source)

	status := &typedSrc.Status

	bucketARN := typedSrc.Spec.ARN

	notifCfg, err := getNotificationsConfig(ctx, cli, bucketARN.Resource)
	switch {
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.AWSS3ReasonNoBucket, "Bucket does not exist")
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"The bucket does not exist: %s", toErrMsg(err)))
	case isAWSError(err):
		// All documented API errors require some user intervention and
		// are not to be retried.
		// https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html
		status.MarkNotSubscribed(v1alpha1.AWSS3ReasonAPIError, "Request to S3 API got rejected")
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Failed to synchronize bucket configuration: %s", toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AWSS3ReasonAPIError, "Cannot obtain current bucket configuration")
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error reading current event notifications configuration: %s", toErrMsg(err)))
	}

	desiredQueueCfg := makeQueueConfiguration(typedSrc, queueARN)

	notifCfg, hasUpdates := setQueueConfiguration(notifCfg, desiredQueueCfg)

	if hasUpdates {
		if err := configureNotifications(ctx, cli, bucketARN.Resource, notifCfg); err != nil {
			status.MarkNotSubscribed(v1alpha1.AWSS3ReasonAPIError, "Cannot configure event notifications")
			return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
				"Error configuring event notifications: %s", toErrMsg(err)))
		}
	}

	if !status.GetCondition(v1alpha1.AWSS3ConditionSubscribed).IsTrue() {
		event.Normal(ctx, ReasonSubscribed, "Configured event notifications for S3 bucket %q", bucketARN)
	}
	status.MarkSubscribed()

	return nil
}

// EnsureNotificationsDisabled ensures that event notifications are disabled in
// the S3 bucket.
func EnsureNotificationsDisabled(ctx context.Context, cli s3iface.S3API) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AWSS3Source)

	bucketARN := typedSrc.Spec.ARN

	notifCfg, err := getNotificationsConfig(ctx, cli, bucketARN.Resource)
	switch {
	case isNotFound(err):
		return reconciler.NewEvent(corev1.EventTypeNormal, ReasonUnsubscribed,
			"Bucket not found, skipping finalization")
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Authorization error getting bucket configuration. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error reading current event notifications configuration: %s", toErrMsg(err))
	}

	notifCfg = removeQueueConfiguration(notifCfg, sourceID(src))

	if err := configureNotifications(ctx, cli, bucketARN.Resource, notifCfg); err != nil {
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error configuring event notifications: %s", toErrMsg(err)))
	}

	return reconciler.NewEvent(corev1.EventTypeNormal, ReasonUnsubscribed,
		"Disabled event notifications for S3 bucket %q", bucketARN)
}

// getNotificationsConfig reads the current event notifications configuration
// of the given S3 bucket.
func getNotificationsConfig(ctx context.Context, cli s3iface.S3API, bucket string) (*s3.NotificationConfiguration, error) {
	resp, err := cli.GetBucketNotificationConfigurationWithContext(ctx, &s3.GetBucketNotificationConfigurationRequest{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("getting configuration: %w", err)
	}

	return resp, nil
}

// configureNotifications configures event notifications for the given S3 bucket.
func configureNotifications(ctx context.Context, cli s3iface.S3API, bucket string, cfg *s3.NotificationConfiguration) error {
	_, err := cli.PutBucketNotificationConfigurationWithContext(ctx, &s3.PutBucketNotificationConfigurationInput{
		Bucket:                    &bucket,
		NotificationConfiguration: cfg,
	})
	if err != nil {
		return fmt.Errorf("setting configuration: %w", err)
	}

	return nil
}

// makeQueueConfiguration returns a QueueConfiguration for the given source.
func makeQueueConfiguration(src *v1alpha1.AWSS3Source, queueARN string) *s3.QueueConfiguration {
	return &s3.QueueConfiguration{
		Id:       aws.String(sourceID(src)),
		Events:   aws.StringSlice(src.Spec.EventTypes),
		QueueArn: &queueARN,
	}
}

// setQueueConfiguration sets/updates a QueueConfiguration in the given
// NotificationConfiguration, without touching existing configurations.
// The returned boolean value indicates whether some updates need to be applied
// to the bucket configuration.
func setQueueConfiguration(nCfg *s3.NotificationConfiguration, qCfg *s3.QueueConfiguration) (*s3.NotificationConfiguration, bool) {
	var isSet bool
	var hasUpdates bool

	for i, cfg := range nCfg.QueueConfigurations {
		if *cfg.Id == *qCfg.Id {
			isSet = true
			nCfg.QueueConfigurations[i] = qCfg
			hasUpdates = !equalEventTypes(qCfg.Events, cfg.Events)
			break
		}
	}
	if !isSet {
		nCfg.QueueConfigurations = append(nCfg.QueueConfigurations, qCfg)
		hasUpdates = true
	}

	return nCfg, hasUpdates
}

// equalEventTypes returns whether two lists of bucket event types are
// semantically equal.
// "b" must be the "current" state, which is expected to always be returned
// sorted by the S3 API, while "a" is user-provided and must be sorted before
// comparing values.
func equalEventTypes(a, b []*string) bool {
	if len(a) != len(b) {
		return false
	}

	sortedA := aws.StringValueSlice(a)
	sort.Strings(sortedA)

	for i := range sortedA {
		if sortedA[i] != *b[i] {
			return false
		}
	}

	return true
}

// removeQueueConfiguration removes a QueueConfiguration by ID from the given
// NotificationConfiguration, without touching other configurations.
func removeQueueConfiguration(nCfg *s3.NotificationConfiguration, id string) *s3.NotificationConfiguration {
	qCfgs := nCfg.QueueConfigurations[:0]

	for _, cfg := range nCfg.QueueConfigurations {
		if *cfg.Id != id {
			qCfgs = append(qCfgs, cfg)
		}
	}

	nCfg.QueueConfigurations = qCfgs

	return nCfg
}

// isNotFound returns whether the given error indicates that some resource was
// not found.
func isNotFound(err error) bool {
	if k8sErr := apierrors.APIStatus(nil); errors.As(err, &k8sErr) {
		return k8sErr.Status().Reason == metav1.StatusReasonNotFound
	}
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		errcode := awsErr.Code()
		return errcode == sqs.ErrCodeQueueDoesNotExist ||
			errcode == s3.ErrCodeNoSuchBucket
	}
	return false
}

// isDenied returns whether the given error indicates that a request to the AWS
// API could not be authorized.
func isDenied(err error) bool {
	if awsErr := awserr.Error(nil); errors.As(err, &awsErr) {
		if awsErr == credentials.ErrStaticCredentialsEmpty {
			return true
		}

		if awsReqFail := awserr.RequestFailure(nil); errors.As(err, &awsReqFail) {
			code := awsReqFail.StatusCode()
			return code == http.StatusUnauthorized || code == http.StatusForbidden
		}
	}
	return false
}

// isAWSError returns whether the given error is an AWS API error.
func isAWSError(err error) bool {
	awsErr := awserr.Error(nil)
	return errors.As(err, &awsErr)
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
