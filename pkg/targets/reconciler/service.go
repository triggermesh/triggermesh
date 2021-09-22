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

package reconciler

import (
	"context"
	"fmt"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/semantic"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// ServiceReconciler performs reconciliation for Services
type ServiceReconciler interface {
	ReconcileService(context.Context, kmeta.OwnerRefableAccessor, *corev1.Service) (*corev1.Service, pkgreconciler.Event)
}

// NewServiceReconciler creates the default implementation for Service reconciler.
func NewServiceReconciler(coreClientSet corev1client.CoreV1Interface, serviceLister corev1listers.ServiceLister) ServiceReconciler {
	return &serviceReconciler{
		coreClientSet: coreClientSet,
		serviceLister: serviceLister,
	}
}

// serviceReconciler performs default reconciliation for Services
type serviceReconciler struct {
	coreClientSet corev1client.CoreV1Interface
	serviceLister corev1listers.ServiceLister
}

// ReconcileService does reconciliation of a desired Service
func (r *serviceReconciler) ReconcileService(ctx context.Context, owner kmeta.OwnerRefableAccessor, expected *corev1.Service) (*corev1.Service, pkgreconciler.Event) {
	s, err := r.findOwned(ctx, owner)
	if apierrors.IsNotFound(err) {
		s, err := r.coreClientSet.Services(expected.Namespace).Create(ctx, expected, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		event.Record(ctx, owner, corev1.EventTypeNormal, "ServiceCreated", `created service: "%s/%s"`, s.Namespace, s.Name)
		return s, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error getting service %q: %v", expected.Name, err)
	}

	// Service owned by the target but with an incorrect name is not expected.
	// If found, delete and let the controller create a new one during the next sync.
	if s.Name != expected.Name {
		logging.FromContext(ctx).Warnf("Deleting Service %s/%s owned by target %s because its name differs "+
			"from expected (%s)", s.Namespace, s.Name, owner.GetObjectMeta().GetName(), expected.Name)

		err := r.coreClientSet.Services(expected.Namespace).Delete(ctx, s.Name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
		event.Record(ctx, owner, corev1.EventTypeWarning, "ServiceDeleted", `deleted service: "%s/%s"`, s.Namespace, s.Name)
		return s, nil
	}

	if !semantic.Semantic.DeepDerivative(expected, s) {
		// resourceVersion must be returned to the API server unmodified for
		// optimistic concurrency, as per Kubernetes API conventions
		expected.ResourceVersion = s.ResourceVersion

		// Preserve status to avoid resetting conditions.
		// Affects only fake Clientsets, necessary for tests.
		expected.Status = s.Status
		expected.Spec.ClusterIP = s.Spec.ClusterIP

		s, err := r.coreClientSet.Services(expected.Namespace).Update(ctx, expected, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
		event.Record(ctx, owner, corev1.EventTypeNormal, "ServiceUpdated", `updated service: "%s/%s"`, s.Namespace, s.Name)
		return s, nil
	}

	return s, nil
}

// findOwned returns a Service owned by the passed object and matched by labels.
func (r *serviceReconciler) findOwned(ctx context.Context, owner kmeta.OwnerRefable) (*corev1.Service, error) {
	sl, err := r.serviceLister.Services(owner.GetObjectMeta().GetNamespace()).List(labels.Everything())
	if err != nil {
		logging.FromContext(ctx).Error("Unable to list services: %v", zap.Error(err))
		return nil, err
	}
	for _, s := range sl {
		if metav1.IsControlledBy(s, owner.GetObjectMeta()) {
			return s, nil
		}
	}

	return nil, apierrors.NewNotFound(corev1.Resource("services"), "")
}
