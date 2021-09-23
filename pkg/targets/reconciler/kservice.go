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

package reconciler

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingclientset "knative.dev/serving/pkg/client/clientset/versioned"
	servinglisters "knative.dev/serving/pkg/client/listers/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/semantic"
)

// List of annotations set on Knative Serving objects by the Knative Serving
// admission webhook.
var knativeServingAnnotations = []string{
	serving.CreatorAnnotation,
	serving.UpdaterAnnotation,
}

// KServiceReconciler performs reconciliation for Knative services
type KServiceReconciler interface {
	ReconcileKService(context.Context, kmeta.OwnerRefable, *servingv1.Service) (*servingv1.Service, pkgreconciler.Event)
}

// NewKServiceReconciler creates the default implementation of KService reconciler.
func NewKServiceReconciler(servingClientSet servingclientset.Interface, servingLister servinglisters.ServiceLister) KServiceReconciler {
	return &kServiceReconciler{
		servingClientSet: servingClientSet,
		servingLister:    servingLister,
	}
}

// newKServiceCreated makes a new reconciler event with event type Normal, and
// reason KServiceCreated.
func newKServiceCreated(namespace, name string) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeNormal, "KServiceCreated", "created kservice: \"%s/%s\"", namespace, name)
}

// newKServiceFailed makes a new reconciler event with event type Warning, and
// reason KServiceFailed.
// FIXME(antoineco): unused
//nolint:golint,unused,deadcode
func newKServiceFailed(namespace, name string, err error) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeWarning, "KServiceFailed", "failed to create kservice: \"%s/%s\", %w", namespace, name, err)
}

// newKServiceUpdated makes a new reconciler event with event type Normal, and
// reason KServiceUpdated.
func newKServiceUpdated(namespace, name string) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeNormal, "KServiceUpdated", "updated kservice: \"%s/%s\"", namespace, name)
}

// newKServiceDeleted makes a new reconciler event with event type Warning, and
// reason KServiceDeleted.
func newKServiceDeleted(namespace, name string) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeWarning, "KServiceDeleted", "deleted kservice: \"%s/%s\"", namespace, name)
}

// kServiceReconciler performs default reconciliation for Knative services
type kServiceReconciler struct {
	servingClientSet servingclientset.Interface
	servingLister    servinglisters.ServiceLister
}

// ReconcileKService does reconciliation of a desired Knative service
func (r *kServiceReconciler) ReconcileKService(ctx context.Context, owner kmeta.OwnerRefable, expected *servingv1.Service) (*servingv1.Service, pkgreconciler.Event) {
	ksvc, err := r.findOwned(ctx, owner)
	if apierrors.IsNotFound(err) {
		ksvc, err = r.servingClientSet.ServingV1().Services(expected.Namespace).Create(ctx, expected, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		return ksvc, newKServiceCreated(ksvc.Namespace, ksvc.Name)
	}

	if err != nil {
		return nil, fmt.Errorf("error getting receive adapter kservice %q: %v", expected.Name, err)
	}

	// Kn service owned by the target but with an incorrect name is not expected.
	// If found, delete and let the controller create a new one during the next sync.
	if ksvc.Name != expected.Name {
		logging.FromContext(ctx).Warnf("Deleting KService %s/%s owned by target %s because its name differs "+
			"from expected (%s)", ksvc.Namespace, ksvc.Name, owner.GetObjectMeta().GetName(), expected.Name)

		err = r.servingClientSet.ServingV1().Services(ksvc.Namespace).Delete(ctx, ksvc.Name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}

		return ksvc, newKServiceDeleted(ksvc.Namespace, ksvc.Name)
	}

	if !semantic.Semantic.DeepEqual(expected, ksvc) {
		// resourceVersion must be returned to the API server unmodified for
		// optimistic concurrency, as per Kubernetes API conventions
		expected.ResourceVersion = ksvc.ResourceVersion

		// immutable Knative annotations must be preserved
		for _, ann := range knativeServingAnnotations {
			if val, ok := ksvc.Annotations[ann]; ok {
				metav1.SetMetaDataAnnotation(&expected.ObjectMeta, ann, val)
			}
		}

		// Preserve status to avoid resetting conditions.
		// Affects only fake Clientsets, necessary for tests.
		expected.Status = ksvc.Status

		ksvc, err = r.servingClientSet.ServingV1().Services(ksvc.Namespace).Update(ctx, expected, metav1.UpdateOptions{})
		if err != nil {
			// TODO send event?
			return nil, err
		}

		return ksvc, newKServiceUpdated(ksvc.Namespace, ksvc.Name)
	}

	return ksvc, nil
}

// findOwned returns a KService owned by the passed object and matched by labels.
func (r *kServiceReconciler) findOwned(ctx context.Context, owner kmeta.OwnerRefable) (*servingv1.Service, error) {
	kl, err := r.servingLister.Services(owner.GetObjectMeta().GetNamespace()).List(labels.Everything())
	if err != nil {
		logging.FromContext(ctx).Error("Unable to list kservices: %v", zap.Error(err))
		return nil, err
	}
	for _, ksvc := range kl {
		if metav1.IsControlledBy(ksvc, owner.GetObjectMeta()) {
			return ksvc, nil
		}
	}
	return nil, apierrors.NewNotFound(servingv1.Resource("services"), "")
}
