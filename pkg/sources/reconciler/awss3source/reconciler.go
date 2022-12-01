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
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/reconciler"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/awss3source"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	s3client "github.com/triggermesh/triggermesh/pkg/sources/client/s3"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	// Getter than can obtain clients for interacting with the S3 and SQS APIs
	s3Cg s3client.ClientGetter

	// SQS adapter
	base       common.GenericDeploymentReconciler[*v1alpha1.AWSS3Source, listersv1alpha1.AWSS3SourceNamespaceLister]
	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// Check that our Reconciler implements Finalizer
var _ reconcilerv1alpha1.Finalizer = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, src *v1alpha1.AWSS3Source) reconciler.Event {
	// inject source into context for usage in reconciliation logic
	ctx = commonv1alpha1.WithReconcilable(ctx, src)

	s3Client, sqsClient, err := r.s3Cg.Get(src)
	if err != nil {
		src.Status.MarkNotSubscribed(v1alpha1.AWSS3ReasonNoClient, "Cannot obtain AWS API clients")
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error creating AWS API clients: %s", err))
	}

	queueARN, err := EnsureQueue(ctx, sqsClient)
	if err != nil {
		return fmt.Errorf("failed to reconcile SQS queue: %w", err)
	}

	if err := r.base.ReconcileAdapter(ctx, r); err != nil {
		return fmt.Errorf("failed to reconcile SQS event source adapter: %w", err)
	}

	return EnsureNotificationsEnabled(ctx, s3Client, queueARN)
}

// FinalizeKind is called when the resource is deleted.
func (r *Reconciler) FinalizeKind(ctx context.Context, src *v1alpha1.AWSS3Source) reconciler.Event {
	// inject source into context for usage in finalization logic
	ctx = commonv1alpha1.WithReconcilable(ctx, src)

	s3Client, sqsClient, err := r.s3Cg.Get(src)
	switch {
	case isNotFound(err):
		// the finalizer is unlikely to recover from a missing Secret,
		// so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Secret missing while finalizing event source. Ignoring: %s", err)
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error creating AWS API clients: %s", err)
	}

	if err := EnsureNoQueue(ctx, sqsClient); err != nil {
		return fmt.Errorf("failed to finalize SQS queue: %w", err)
	}

	// The finalizer blocks the deletion of the source object until
	// ensureNotificationsDisabled succeeds to ensure that we don't leave
	// any dangling event notification configurations behind us.
	return EnsureNotificationsDisabled(ctx, s3Client)
}

// sourceID returns an ID that identifies the given source instance in AWS
// resources or resources tags.
func sourceID(src commonv1alpha1.Reconcilable) string {
	return "io.triggermesh.awss3sources." + src.GetNamespace() + "." + src.GetName()
}
