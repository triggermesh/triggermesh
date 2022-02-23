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

package tektontarget

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/ptr"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

const tektontargetServiceAccountName = "tektontarget-adapter"

// reconcileServiceAccounts ensures that ServiceAccounts (and their
// corresponding RoleBinding) pertaining to TektonTarget objects are present or
// absent from the given namespace, depending on the TektonTarget objects
// present in that same namespace.
func (r *reconciler) reconcileServiceAccounts(ctx context.Context, ns string) error {
	tts, err := r.targetLister(ns).List(labels.Everything())
	if err != nil {
		return err
	}

	if len(tts) == 0 {
		// The ServiceAccount is garbage collected as soon as its last
		// owner (TektonTarget) is deleted, so there is no need to
		// explicitly delete the objects we may have previously created.
		return nil
	}
	return r.ensureAdapterServiceAccount(ctx, newServiceAccount(ns, tts))
}

// ensureAdapterServiceAccount ensures a ServiceAccount and its corresponding
// RoleBinding exist for the target's adapter.
func (r *reconciler) ensureAdapterServiceAccount(ctx context.Context, desiredSA *corev1.ServiceAccount) error {
	currentSA, err := r.getOrCreateAdapterServiceAccount(ctx, desiredSA)
	if err != nil {
		return err
	}

	desiredRB := newRoleBinding(currentSA)

	currentRB, err := r.getOrCreateAdapterRoleBinding(ctx, desiredRB)
	if err != nil {
		return err
	}

	if _, err = r.syncAdapterServiceAccount(ctx, currentSA, desiredSA); err != nil {
		return fmt.Errorf("synchronizing adapter ServiceAccount: %w", err)
	}

	if _, err = r.syncAdapterRoleBinding(ctx, currentRB, desiredRB); err != nil {
		return fmt.Errorf("synchronizing adapter RoleBinding: %w", err)
	}

	return nil
}

// getOrCreateAdapterServiceAccount returns the existing adapter
// ServiceAccount, or creates it if it is missing.
func (r *reconciler) getOrCreateAdapterServiceAccount(ctx context.Context,
	desiredSA *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {

	currentSA, err := r.saLister(desiredSA.Namespace).Get(desiredSA.Name)
	switch {
	case apierrors.IsNotFound(err):
		currentSA, err = r.saClient(desiredSA.Namespace).Create(ctx, desiredSA, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("creating adapter ServiceAccount: %w", err)
		}
		controller.GetEventRecorder(ctx).Eventf(newNamespace(desiredSA.Namespace), corev1.EventTypeNormal,
			"Created", "ServiceAccount %q created due to the creation of a TektonTarget", currentSA.Name)

	case err != nil:
		return nil, fmt.Errorf("getting adapter ServiceAccount from cache: %w", err)
	}

	return currentSA, nil
}

// syncAdapterServiceAccount synchronizes the desired state of an adapter
// ServiceAccount against its current state in the running cluster.
func (r *reconciler) syncAdapterServiceAccount(ctx context.Context,
	currentSA, desiredSA *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {

	if reflect.DeepEqual(desiredSA.OwnerReferences, currentSA.OwnerReferences) {
		return currentSA, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredSA.ResourceVersion = currentSA.ResourceVersion

	currentSA, err := r.saClient(desiredSA.Namespace).Update(ctx, desiredSA, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating adapter ServiceAccount: %w", err)
	}

	controller.GetEventRecorder(ctx).Eventf(newNamespace(desiredSA.Namespace), corev1.EventTypeNormal,
		"Updated", "ServiceAccount %q updated due to the creation/deletion of a TektonTarget", currentSA.Name)

	return currentSA, nil
}

// getOrCreateAdapterRoleBinding returns the existing adapter RoleBinding, or
// creates it if it is missing.
func (r *reconciler) getOrCreateAdapterRoleBinding(ctx context.Context,
	desiredRB *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {

	currentSA, err := r.rbLister(desiredRB.Namespace).Get(desiredRB.Name)
	switch {
	case apierrors.IsNotFound(err):
		currentSA, err = r.rbClient(desiredRB.Namespace).Create(ctx, desiredRB, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("creating adapter RoleBinding: %w", err)
		}
		controller.GetEventRecorder(ctx).Eventf(newNamespace(desiredRB.Namespace), corev1.EventTypeNormal,
			"Created", "RoleBinding %q created due to the creation of a TektonTarget", currentSA.Name)

	case err != nil:
		return nil, fmt.Errorf("getting adapter RoleBinding from cache: %w", err)
	}

	return currentSA, nil
}

// syncAdapterRoleBinding synchronizes the desired state of an adapter
// RoleBinding against its current state in the running cluster.
func (r *reconciler) syncAdapterRoleBinding(ctx context.Context,
	currentRB, desiredRB *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {

	if reflect.DeepEqual(desiredRB.OwnerReferences, currentRB.OwnerReferences) &&
		reflect.DeepEqual(desiredRB.RoleRef, currentRB.RoleRef) &&
		reflect.DeepEqual(desiredRB.Subjects, currentRB.Subjects) {

		return currentRB, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredRB.ResourceVersion = currentRB.ResourceVersion

	currentRB, err := r.rbClient(desiredRB.Namespace).Update(ctx, desiredRB, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating adapter RoleBinding: %w", err)
	}
	controller.GetEventRecorder(ctx).Eventf(newNamespace(desiredRB.Namespace), corev1.EventTypeNormal,
		"Updated", "RoleBinding %q updated", currentRB.Name)

	return currentRB, nil
}

func newServiceAccount(ns string, tts []*v1alpha1.TektonTarget) *corev1.ServiceAccount {
	gvk := (*v1alpha1.TektonTarget)(nil).GetGroupVersionKind()

	ownerRefs := make([]metav1.OwnerReference, len(tts))
	for i, tt := range tts {
		ownerRefs[i] = *metav1.NewControllerRef(tt, gvk)
		ownerRefs[i].Controller = ptr.Bool(false)
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       ns,
			Name:            tektontargetServiceAccountName,
			Labels:          libreconciler.MakeAdapterLabels(adapterName, nil),
			OwnerReferences: ownerRefs,
		},
	}

}

func newRoleBinding(owner *corev1.ServiceAccount) *rbacv1.RoleBinding {
	crGVK := rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
	saGVK := corev1.SchemeGroupVersion.WithKind("ServiceAccount")

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.Namespace,
			Name:      tektontargetServiceAccountName,
			Labels:    libreconciler.MakeAdapterLabels(adapterName, nil),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(owner, saGVK),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: crGVK.Group,
			Kind:     crGVK.Kind,
			Name:     tektontargetServiceAccountName,
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  saGVK.Group,
			Kind:      saGVK.Kind,
			Name:      tektontargetServiceAccountName,
			Namespace: owner.Namespace,
		}},
	}
}

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
