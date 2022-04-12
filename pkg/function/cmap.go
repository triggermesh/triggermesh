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

package function

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/tracker"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const functionNameLabel = "extensions.triggermesh.io/function"

const codeCmapDataKey = "code"

// appInstanceLabel is a unique name identifying the instance of an application.
// See Kubernetes recommended labels
// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
const appInstanceLabel = "app.kubernetes.io/instance"

var configMapGVK = corev1.SchemeGroupVersion.WithKind("ConfigMap")

func (r *Reconciler) reconcileConfigmap(ctx context.Context) error {
	f := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.Function)
	status := &f.Status

	desiredCmap := newCodeConfigMap(f)

	currentCmap, err := r.getOrCreateCodeConfigMap(ctx, desiredCmap)
	if err != nil {
		status.MarkConfigMapUnavailable(v1alpha1.FunctionReasonFailedSync, fmt.Sprintf(
			"Failed to get or create code ConfigMap: %s", err))
		return err
	}

	if currentCmap, err = r.syncCodeConfigMap(ctx, currentCmap, desiredCmap); err != nil {
		status.MarkConfigMapUnavailable(v1alpha1.FunctionReasonFailedSync, fmt.Sprintf(
			"Failed to synchronize code ConfigMap: %s", err))
		return fmt.Errorf("synchronizing code ConfigMap: %w", err)
	}

	cmRef := tracker.Reference{
		APIVersion: configMapGVK.GroupVersion().String(),
		Kind:       configMapGVK.Kind,
		Name:       currentCmap.Name,
		Namespace:  currentCmap.Namespace,
	}

	if err := r.tracker.TrackReference(cmRef, f); err != nil {
		return fmt.Errorf("tracking changes to code ConfigMap: %w", err)
	}

	status.MarkConfigMapAvailable(currentCmap.Name, currentCmap.ResourceVersion)

	return nil
}

// getOrCreateCodeConfigMap returns the existing code ConfigMap for a given
// Function instance, or creates it if it is missing.
func (r *Reconciler) getOrCreateCodeConfigMap(ctx context.Context,
	desiredCmap *corev1.ConfigMap) (*corev1.ConfigMap, error) {

	rcl := commonv1alpha1.ReconcilableFromContext(ctx)

	cmap, err := r.findCodeConfigMap(rcl, metav1.GetControllerOfNoCopy(desiredCmap))
	switch {
	case apierrors.IsNotFound(err):
		cmap, err = r.cmCli(desiredCmap.Namespace).Create(ctx, desiredCmap, metav1.CreateOptions{})
		if err != nil {
			return nil, reconciler.NewEvent(corev1.EventTypeWarning, "FailedConfigMapCreate",
				"Failed to create code ConfigMap %q: %s", desiredCmap.Name, err)
		}
		event.Normal(ctx, "CreateConfigMap", "Created code ConfigMap %q", cmap.Name)

	case err != nil:
		return nil, fmt.Errorf("getting code ConfigMap from cache: %w", err)
	}

	return cmap, nil
}

// syncCodeConfigMap synchronizes the desired state of a Function's code
// ConfigMap against its current state in the running cluster.
func (r *Reconciler) syncCodeConfigMap(ctx context.Context,
	currentCmap, desiredCmap *corev1.ConfigMap) (*corev1.ConfigMap, error) {

	if reflect.DeepEqual(desiredCmap.Data, currentCmap.Data) {
		return currentCmap, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredCmap.ResourceVersion = currentCmap.ResourceVersion

	cmap, err := r.cmCli(desiredCmap.Namespace).Update(ctx, desiredCmap, metav1.UpdateOptions{})
	if err != nil {
		return nil, reconciler.NewEvent(corev1.EventTypeWarning, "FailedConfigMapUpdate",
			"Failed to update code ConfigMap %q: %s", desiredCmap.Name, err)
	}
	event.Normal(ctx, "UpdateConfigMap", "Updated code ConfigMap %q", cmap.Name)

	return cmap, nil
}

// findCodeConfigMap returns the ConfigMap containing the code of the given
// Function instance if it exists.
func (r *Reconciler) findCodeConfigMap(rcl commonv1alpha1.Reconcilable,
	owner *metav1.OwnerReference) (*corev1.ConfigMap, error) {

	ls := common.CommonObjectLabels(rcl)

	// the combination of standard labels {name,instance} is unique
	// and immutable for single-tenant components
	ls[appInstanceLabel] = rcl.GetName()

	sel := labels.SelectorFromValidatedSet(ls)

	cmaps, err := r.cmLister(rcl.GetNamespace()).List(sel)
	if err != nil {
		return nil, err
	}

	for _, cmap := range cmaps {
		cmapOwner := metav1.GetControllerOfNoCopy(cmap)

		if cmapOwner.UID == owner.UID {
			return cmap, nil
		}
	}

	gr := corev1.Resource("configmaps")

	return nil, newNotFoundForSelector(gr, sel)
}

// newNotFoundForSelector returns an error which indicates that no object of
// the type matching the given GroupResource was found for the given label
// selector.
func newNotFoundForSelector(gr schema.GroupResource, sel labels.Selector) *apierrors.StatusError {
	err := apierrors.NewNotFound(gr, "")
	err.ErrStatus.Message = fmt.Sprint(gr, " not found for selector ", sel)
	return err
}

// newCodeConfigMap returns a ConfigMap object containing the code of the given Function.
func newCodeConfigMap(f *v1alpha1.Function) *corev1.ConfigMap {
	ns := f.Namespace
	name := f.Name

	return resource.NewConfigMap(ns, kmeta.ChildName(common.ComponentName(f)+"-code-", name),
		resource.Controller(f),

		resource.Labels(common.CommonObjectLabels(f)),
		resource.Label(appInstanceLabel, name),
		resource.Label(functionNameLabel, name),

		resource.Data(codeCmapDataKey, f.Spec.Code),
	)
}
