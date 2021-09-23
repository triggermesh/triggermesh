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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	appsv1listers "k8s.io/client-go/listers/apps/v1"

	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/semantic"
)

// DeploymentReconciler performs reconciliation for Deployments
type DeploymentReconciler interface {
	ReconcileDeployment(context.Context, kmeta.OwnerRefableAccessor, *appsv1.Deployment) (*appsv1.Deployment, error)
}

// NewDeploymentReconciler creates the default implementation for Deployment reconciler.
func NewDeploymentReconciler(appsClientSet appsv1client.AppsV1Interface, deploymentLister appsv1listers.DeploymentLister) DeploymentReconciler {
	return &deploymentReconciler{
		appsClientSet:    appsClientSet,
		deploymentLister: deploymentLister,
	}
}

// serviceReconciler performs default reconciliation for Deployments
type deploymentReconciler struct {
	appsClientSet    appsv1client.AppsV1Interface
	deploymentLister appsv1listers.DeploymentLister
}

func (r *deploymentReconciler) ReconcileDeployment(ctx context.Context, owner kmeta.OwnerRefableAccessor, expected *appsv1.Deployment) (*appsv1.Deployment, error) {
	d, err := r.findOwned(ctx, owner)
	if apierrors.IsNotFound(err) {
		d, err := r.appsClientSet.Deployments(expected.Namespace).Create(ctx, expected, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		event.Record(ctx, owner, corev1.EventTypeNormal, "DeploymentCreated", `created deployment: "%s/%s"`, d.Namespace, d.Name)
		return d, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error getting deployment %q: %v", expected.Name, err)
	}

	// Deployment owned by the target but with an incorrect name is not expected.
	// If found, delete and let the controller create a new one during the next sync.
	if d.Name != expected.Name {
		logging.FromContext(ctx).Warnf("Deleting Deployment %s/%s owned by target %s because its name differs "+
			"from expected (%s)", d.Namespace, d.Name, owner.GetObjectMeta().GetName(), expected.Name)

		err := r.appsClientSet.Deployments(expected.Namespace).Delete(ctx, d.Name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
		event.Record(ctx, owner, corev1.EventTypeWarning, "DeploymentDeleted", `deleted deployment: "%s/%s"`, d.Namespace, d.Name)
		return d, nil
	}

	if semantic.Semantic.DeepEqual(expected, d) {
		return d, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	expected.ResourceVersion = d.ResourceVersion

	// Preserve status to avoid resetting conditions.
	// Affects only fake Clientsets, necessary for tests.
	expected.Status = d.Status

	d, err = r.appsClientSet.Deployments(expected.Namespace).Update(ctx, expected, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	event.Record(ctx, owner, corev1.EventTypeNormal, "DeploymentUpdated", `updated deployment: "%s/%s"`, d.Namespace, d.Name)
	return d, nil
}

// findOwned returns a Deployment owned by the passed object and matched by labels.
func (r *deploymentReconciler) findOwned(ctx context.Context, owner kmeta.OwnerRefable) (*appsv1.Deployment, error) {
	dl, err := r.deploymentLister.Deployments(owner.GetObjectMeta().GetNamespace()).List(labels.Everything())
	if err != nil {
		logging.FromContext(ctx).Error("Unable to list deployments: %v", zap.Error(err))
		return nil, err
	}
	for _, d := range dl {
		if metav1.IsControlledBy(d, owner.GetObjectMeta()) {
			return d, nil
		}
	}

	return nil, apierrors.NewNotFound(appsv1.Resource("deployments"), "")
}
