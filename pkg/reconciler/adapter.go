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

package reconciler

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/system"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const (
	metricsPrometheusPortName = "metrics"

	metricsPrometheusPort uint16 = 9090

	// TCP port used to expose metrics via the Prometheus metrics exporter in
	// components backed by a Knative Service.
	// It is necessary to override Knative's default value of "9090" because this
	// port is already reserved by the "queue-proxy" container in Knative Services.
	metricsPrometheusPortKsvc uint16 = 9092
)

// ComponentName returns the component name for the given object.
func ComponentName(o kmeta.OwnerRefable) string {
	return strings.ToLower(o.GetGroupVersionKind().Kind)
}

// MTAdapterObjectName returns a unique name to apply to all objects related to
// the given component's multi-tenant adapter (RBAC, Deployment/KnService, ...).
func MTAdapterObjectName(o kmeta.OwnerRefable) string {
	return ComponentName(o) + "-" + componentAdapter
}

// ServiceAccountName returns the name to set on the ServiceAccount associated
// with the given component instance.
func ServiceAccountName(rcl v1alpha1.Reconcilable) string {
	if v1alpha1.WantsOwnServiceAccount(rcl) {
		rclName := rcl.GetName()

		// Edge case: we need to make sure some characters are inserted
		// between the component name and the component instance's name
		// to avoid clashing with the shared "{kind}-adapter"
		// ServiceAccount in case the component instance is named
		// "adapter". We picked 'i' for "instance" to keep it short yet
		// distinguishable.
		return kmeta.ChildName(ComponentName(rcl)+"-i-", rclName)
	}

	return MTAdapterObjectName(rcl)
}

// NewAdapterDeployment is a wrapper around resource.NewDeployment which
// pre-populates attributes common to all adapters backed by a Deployment.
func NewAdapterDeployment(rcl v1alpha1.Reconcilable, sinkURI *apis.URL, opts ...resource.ObjectOption) *appsv1.Deployment {
	rclNs := rcl.GetNamespace()
	rclName := rcl.GetName()

	var sinkURIStr string
	if sinkURI != nil {
		sinkURIStr = sinkURI.String()
	}

	return resource.NewDeployment(rclNs, kmeta.ChildName(ComponentName(rcl)+"-", rclName),
		append(commonAdapterDeploymentOptions(rcl), append([]resource.ObjectOption{
			resource.Controller(rcl),

			resource.Label(appInstanceLabel, rclName),
			resource.Selector(appInstanceLabel, rclName),

			resource.EnvVar(envSink, sinkURIStr),
		}, opts...)...)...,
	)
}

// NewMTAdapterDeployment is a wrapper around resource.NewDeployment which
// pre-populates attributes common to all multi-tenant adapters backed by a
// Deployment.
func NewMTAdapterDeployment(rcl v1alpha1.Reconcilable, opts ...resource.ObjectOption) *appsv1.Deployment {
	rclNs := rcl.GetNamespace()

	return resource.NewDeployment(rclNs, MTAdapterObjectName(rcl),
		append(commonAdapterDeploymentOptions(rcl), append([]resource.ObjectOption{
			resource.EnvVar(EnvNamespace, rclNs),
			resource.EnvVar(system.NamespaceEnvKey, rclNs), // required to enable HA
		}, opts...)...)...,
	)
}

// commonAdapterDeploymentOptions returns a set of ObjectOptions common to all
// adapters backed by a Deployment.
func commonAdapterDeploymentOptions(rcl v1alpha1.Reconcilable) []resource.ObjectOption {
	app := ComponentName(rcl)

	objectOptions := []resource.ObjectOption{
		resource.TerminationErrorToLogs,

		resource.Label(appNameLabel, app),
		resource.Label(appComponentLabel, componentAdapter),
		resource.Label(appPartOfLabel, partOf),
		resource.Label(appManagedByLabel, managedBy),

		resource.Selector(appNameLabel, app),
		resource.Selector(appComponentLabel, componentAdapter),
		resource.PodLabel(appPartOfLabel, partOf),
		resource.PodLabel(appManagedByLabel, managedBy),

		resource.ServiceAccount(ServiceAccountName(rcl)),

		resource.EnvVar(envComponent, app),

		resource.Port(metricsPrometheusPortName, int32(metricsPrometheusPort)),
	}

	parentLabels := rcl.GetLabels()
	for _, key := range labelsPropagationList {
		if value, exists := parentLabels[key]; exists {
			objectOptions = append(objectOptions, resource.Label(key, value))
			objectOptions = append(objectOptions, resource.PodLabel(key, value))
		}
	}

	if cfbl, canConfigureAdapter := rcl.(v1alpha1.AdapterConfigurable); canConfigureAdapter {
		if overrides := cfbl.GetAdapterOverrides(); overrides != nil {
			objectOptions = append(objectOptions, adapterOverrideOptions(overrides)...)
		}
	}

	return objectOptions
}

// NewAdapterKnService is a wrapper around resource.NewKnService which
// pre-populates attributes common to all adapters backed by a Knative Service.
func NewAdapterKnService(rcl v1alpha1.Reconcilable, sinkURI *apis.URL, opts ...resource.ObjectOption) *servingv1.Service {
	rclNs := rcl.GetNamespace()
	rclName := rcl.GetName()

	var sinkURIStr string
	if sinkURI != nil {
		sinkURIStr = sinkURI.String()
	}

	return resource.NewKnService(rclNs, kmeta.ChildName(ComponentName(rcl)+"-", rclName),
		append(commonAdapterKnServiceOptions(rcl), append([]resource.ObjectOption{
			resource.Controller(rcl),

			resource.EnvVar(EnvNamespace, rclNs),
			resource.EnvVar(EnvName, rclName),

			resource.Label(appInstanceLabel, rclName),
			resource.PodLabel(appInstanceLabel, rclName),

			resource.EnvVar(envSink, sinkURIStr),
		}, opts...)...)...,
	)
}

// NewMTAdapterKnService is a wrapper around resource.NewKnService which
// pre-populates attributes common to all multi-tenant adapters backed by a
// Knative Service.
func NewMTAdapterKnService(rcl v1alpha1.Reconcilable, opts ...resource.ObjectOption) *servingv1.Service {
	rclNs := rcl.GetNamespace()

	return resource.NewKnService(rclNs, MTAdapterObjectName(rcl),
		append(commonAdapterKnServiceOptions(rcl), append([]resource.ObjectOption{
			resource.EnvVar(EnvNamespace, rclNs),
			resource.EnvVar(system.NamespaceEnvKey, rclNs), // required to enable HA
		}, opts...)...)...,
	)
}

// commonAdapterKnServiceOptions returns a set of ObjectOptions common to all
// adapters backed by a Knative Service.
func commonAdapterKnServiceOptions(rcl v1alpha1.Reconcilable) []resource.ObjectOption {
	app := ComponentName(rcl)

	objectOptions := []resource.ObjectOption{
		resource.VisibilityClusterLocal,

		resource.Label(appNameLabel, app),
		resource.Label(appComponentLabel, componentAdapter),
		resource.Label(appPartOfLabel, partOf),
		resource.Label(appManagedByLabel, managedBy),

		resource.PodLabel(appNameLabel, app),
		resource.PodLabel(appComponentLabel, componentAdapter),
		resource.PodLabel(appPartOfLabel, partOf),
		resource.PodLabel(appManagedByLabel, managedBy),

		resource.ServiceAccount(MTAdapterObjectName(rcl)),

		resource.EnvVar(envComponent, app),
		resource.EnvVar(envMetricsPrometheusPort, strconv.FormatUint(uint64(metricsPrometheusPortKsvc), 10)),
	}

	parentLabels := rcl.GetLabels()
	for _, key := range labelsPropagationList {
		if value, exists := parentLabels[key]; exists {
			objectOptions = append(objectOptions, resource.Label(key, value))
			objectOptions = append(objectOptions, resource.PodLabel(key, value))
		}
	}

	if cfbl, canConfigureAdapter := rcl.(v1alpha1.AdapterConfigurable); canConfigureAdapter {
		if overrides := cfbl.GetAdapterOverrides(); overrides != nil {
			objectOptions = append(objectOptions, adapterOverrideOptions(overrides)...)
		}
	}

	return objectOptions
}

// newServiceAccount returns a ServiceAccount object with its OwnerReferences
// metadata attribute populated from the given owners.
func newServiceAccount(rcl v1alpha1.Reconcilable, owners []kmeta.OwnerRefable) *corev1.ServiceAccount {
	return resource.NewServiceAccount(rcl.GetNamespace(), ServiceAccountName(rcl),
		resource.Owners(owners...),
		resource.Labels(CommonObjectLabels(rcl)),
	)
}

// newRoleBinding returns a RoleBinding object that binds a ServiceAccount
// (namespace-scoped) to a ClusterRole (cluster-scoped).
func newRoleBinding(rcl v1alpha1.Reconcilable, owner *corev1.ServiceAccount) *rbacv1.RoleBinding {
	crGVK := rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
	saGVK := corev1.SchemeGroupVersion.WithKind("ServiceAccount")

	ns := rcl.GetNamespace()
	n := MTAdapterObjectName(rcl)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      n,
			Labels:    CommonObjectLabels(rcl),
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
// objects reconciled for the given component type.
func CommonObjectLabels(o kmeta.OwnerRefable) labels.Set {
	return labels.Set{
		appNameLabel:      ComponentName(o),
		appComponentLabel: componentAdapter,
		appPartOfLabel:    partOf,
		appManagedByLabel: managedBy,
	}
}

// MaybeAppendValueFromEnvVar conditionally appends an EnvVar to env based on
// the contents of valueFrom.
// ValueFromSecret takes precedence over Value in case the API didn't reject
// the object despite the CRD's schema validation
func MaybeAppendValueFromEnvVar(envs []corev1.EnvVar, key string, valueFrom v1alpha1.ValueFromField) []corev1.EnvVar {
	if vfs := valueFrom.ValueFromSecret; vfs != nil {
		return append(envs, corev1.EnvVar{
			Name: key,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: vfs,
			},
		})
	}

	if v := valueFrom.Value; v != "" {
		return append(envs, corev1.EnvVar{
			Name:  key,
			Value: v,
		})
	}

	return envs
}

// adapterOverrideOptions applies adapter override parameters depending on
// deployment type.
func adapterOverrideOptions(overrides *v1alpha1.AdapterOverrides) []resource.ObjectOption {
	var opts []resource.ObjectOption

	opts = append(opts, func(object interface{}) {
		if _, ok := object.(*servingv1.Service); ok {
			if public := overrides.Public; public != nil && *public {
				resource.VisibilityPublic(object)
			} else {
				resource.VisibilityClusterLocal(object)
			}
		}
	})

	if overrides.Resources != nil {
		opts = append(opts, resource.Requests(toQuantity(overrides.Resources.Requests)))
		opts = append(opts, resource.Limits(toQuantity(overrides.Resources.Limits)))
	}

	for _, t := range overrides.Tolerations {
		opts = append(opts, resource.Toleration(t))
	}

	return opts
}

// toQuantity converts corev1.ResourceList to separate CPU and Memory quantities.
func toQuantity(resources corev1.ResourceList) (*kres.Quantity, *kres.Quantity) {
	var cpu, memory *kres.Quantity

	for k, v := range resources {
		switch k {
		case corev1.ResourceCPU:
			cpu = &v
		case corev1.ResourceMemory:
			memory = &v
		}
	}
	return cpu, memory
}

// RBACOwners helps implementing RBACOwnersLister generically.
func RBACOwners[T kmeta.OwnerRefable](l Lister[T]) ([]kmeta.OwnerRefable, error) {
	ts, err := l.List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(ts))
	for i := range ts {
		ownerRefables[i] = ts[i]
	}

	return ownerRefables, nil
}
