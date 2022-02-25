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

package common

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	network "knative.dev/networking/pkg"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/system"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/routing/reconciler/common/resource"
)

const metricsPrometheusPortKsvc uint16 = 9092

// ComponentName returns the component name for the given source object.
func ComponentName(rtr kmeta.OwnerRefable) string {
	return strings.ToLower(rtr.GetGroupVersionKind().Kind)
}

// MTAdapterObjectName returns a unique name to apply to all objects related to
// the given source's multi-tenant adapter (RBAC, KnService, ...).
func MTAdapterObjectName(rtr kmeta.OwnerRefable) string {
	return ComponentName(rtr) + "-" + componentAdapter
}

// NewMTAdapterKnService is a wrapper around resource.NewKnService which
// pre-populates attributes common to all multi-tenant adapters backed by a
// Knative Service.
func NewMTAdapterKnService(rtr v1alpha1.Router, opts ...resource.ObjectOption) *servingv1.Service {
	rtrNs := rtr.GetNamespace()

	return resource.NewKnService(rtrNs, MTAdapterObjectName(rtr),
		append(commonAdapterKnServiceOptions(rtr), append([]resource.ObjectOption{
			resource.EnvVar(EnvNamespace, rtrNs),
			resource.EnvVar(system.NamespaceEnvKey, rtrNs), // required to enable HA
		}, opts...)...)...,
	)
}

// commonAdapterKnServiceOptions returns a set of ObjectOptions common to all
// adapters backed by a Knative Service.
func commonAdapterKnServiceOptions(rtr v1alpha1.Router) []resource.ObjectOption {
	app := ComponentName(rtr)

	objectOptions := []resource.ObjectOption{
		resource.Label(appNameLabel, app),
		resource.Label(appComponentLabel, componentAdapter),
		resource.Label(appPartOfLabel, partOf),
		resource.Label(appManagedByLabel, managedBy),
		resource.Label(network.VisibilityLabelKey, serving.VisibilityClusterLocal),

		resource.PodLabel(appNameLabel, app),
		resource.PodLabel(appComponentLabel, componentAdapter),
		resource.PodLabel(appPartOfLabel, partOf),
		resource.PodLabel(appManagedByLabel, managedBy),

		resource.ServiceAccount(MTAdapterObjectName(rtr)),

		resource.EnvVar(envComponent, app),
		resource.EnvVar(envMetricsPrometheusPort, strconv.FormatUint(uint64(metricsPrometheusPortKsvc), 10)),
	}

	parentLabels := rtr.GetLabels()
	for _, key := range labelsPropagationList {
		if value, exists := parentLabels[key]; exists {
			objectOptions = append(objectOptions, resource.Label(key, value))
			objectOptions = append(objectOptions, resource.PodLabel(key, value))
		}
	}

	return objectOptions
}

// newServiceAccount returns a ServiceAccount object with its OwnerReferences
// metadata attribute populated from the given owners.
func newServiceAccount(rtr v1alpha1.Router, owners []kmeta.OwnerRefable) *corev1.ServiceAccount {
	ownerRefs := make([]metav1.OwnerReference, len(owners))
	for i, owner := range owners {
		ownerRefs[i] = *kmeta.NewControllerRef(owner)
		ownerRefs[i].Controller = ptr.Bool(false)
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       rtr.GetNamespace(),
			Name:            MTAdapterObjectName(rtr),
			OwnerReferences: ownerRefs,
			Labels:          CommonObjectLabels(rtr),
		},
	}

}

// newRoleBinding returns a RoleBinding object that binds a ServiceAccount
// (namespace-scoped) to a ClusterRole (cluster-scoped).
func newRoleBinding(rtr v1alpha1.Router, owner *corev1.ServiceAccount) *rbacv1.RoleBinding {
	crGVK := rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
	saGVK := corev1.SchemeGroupVersion.WithKind("ServiceAccount")

	ns := rtr.GetNamespace()
	n := MTAdapterObjectName(rtr)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      n,
			Labels:    CommonObjectLabels(rtr),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: crGVK.Group,
			Kind:     crGVK.Kind,
			Name:     n,
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  saGVK.Group,
			Kind:      saGVK.Kind,
			Namespace: ns,
			Name:      n,
		}},
	}

	OwnByServiceAccount(rb, owner)

	return rb
}

// OwnByServiceAccount sets the owner of obj to the given ServiceAccount.
func OwnByServiceAccount(obj metav1.Object, owner *corev1.ServiceAccount) {
	saGVK := corev1.SchemeGroupVersion.WithKind("ServiceAccount")

	obj.SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(owner, saGVK),
	})
}

// CommonObjectLabels returns a set of labels which are always applied to
// objects reconciled for the given source type.
func CommonObjectLabels(rtr kmeta.OwnerRefable) labels.Set {
	return labels.Set{
		appNameLabel:      ComponentName(rtr),
		appComponentLabel: componentAdapter,
		appPartOfLabel:    partOf,
		appManagedByLabel: managedBy,
	}
}
