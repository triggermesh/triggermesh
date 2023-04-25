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
	"strconv"
	"strings"
	"time"

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

	// TCP port used to expose metrics via the Prometheus metrics exporter
	// in TriggerMesh components.
	//
	// It is necessary to override Knative's default value of "9090"
	// because this port is already reserved by the "queue-proxy" container
	// in Knative Services.
	// For consistency, we use the same port in components that aren't
	// backed by a Knative Service.
	metricsPrometheusPort uint16 = 9092
)

const roleNameConfigWatcher = "triggermesh-config-watcher"

const defaultSinkTimeout = 30 * time.Second

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

			// Used to label Prometheus metrics with the component
			// instance's namespace and name
			resource.EnvVar(EnvNamespace, rclNs),
			resource.EnvVar(EnvName, rclName),

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
			// makes Informers namespace-scoped in TriggerMesh's adapter shared Main
			resource.EnvVar(EnvNamespace, rclNs),
			// required to enable Knative's HA
			resource.EnvVar(system.NamespaceEnvKey, rclNs),
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

		resource.ServiceAccount(serviceAccount(rcl)),

		resource.EnvVar(envComponent, app),
		resource.EnvVar(envSinkTimeout, strconv.FormatInt(int64(defaultSinkTimeout.Seconds()), 10)),
		resource.EnvVar(envMetricsPrometheusPort, strconv.FormatUint(uint64(metricsPrometheusPort), 10)),

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

			// Used to label Prometheus metrics with the component
			// instance's namespace and name
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
			// makes Informers namespace-scoped in TriggerMesh's adapter shared Main
			resource.EnvVar(EnvNamespace, rclNs),
			// required to enable Knative's HA
			resource.EnvVar(system.NamespaceEnvKey, rclNs),
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

		resource.ServiceAccount(serviceAccount(rcl)),

		resource.EnvVar(envComponent, app),
		resource.EnvVar(envSinkTimeout, strconv.FormatInt(int64(defaultSinkTimeout.Seconds()), 10)),
		resource.EnvVar(envMetricsPrometheusPort, strconv.FormatUint(uint64(metricsPrometheusPort), 10)),
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

// serviceAccount returns a ServiceAccount object with its OwnerReferences
// metadata attribute populated from the given owners.
func serviceAccount(rcl v1alpha1.Reconcilable, owners ...kmeta.OwnerRefable) *corev1.ServiceAccount {
	sa := resource.NewServiceAccount(rcl.GetNamespace(), ServiceAccountName(rcl),
		resource.Owners(owners...),
		resource.Labels(CommonObjectLabels(rcl)),
	)
	for _, m := range serviceAccountMutations(rcl) {
		m(sa)
	}
	return sa
}

// newConfigWatchRoleBinding returns a RoleBinding object that binds a ServiceAccount
// (namespace-scoped) to the config watcher ClusterRole (cluster-scoped).
func newConfigWatchRoleBinding(rcl v1alpha1.Reconcilable, owner *corev1.ServiceAccount) *rbacv1.RoleBinding {
	rbName := owner.Name + "-config-watcher" // {kind}-adapter-config-watcher or {kind}-i-{name}-config-watcher

	return newRoleBinding(rbName, roleNameConfigWatcher, rcl, owner)
}

// newMTAdapterRoleBinding returns a RoleBinding object that binds a ServiceAccount
// (namespace-scoped) to the (mt-)adapter's ClusterRole (cluster-scoped).
func newMTAdapterRoleBinding(rcl v1alpha1.Reconcilable, owner *corev1.ServiceAccount) *rbacv1.RoleBinding {
	// Per convention, both the roleBinding and clusterRole of multi-tenant
	// adapters are named after the component type.
	baseName := MTAdapterObjectName(rcl) // {kind}-adapter

	return newRoleBinding(baseName, baseName, rcl, owner)
}

// newRoleBinding returns a RoleBinding object that binds a ServiceAccount
// (namespace-scoped) to a ClusterRole (cluster-scoped).
func newRoleBinding(name, roleName string, rcl v1alpha1.Reconcilable, owner *corev1.ServiceAccount) *rbacv1.RoleBinding {
	crGVK := rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
	saGVK := corev1.SchemeGroupVersion.WithKind("ServiceAccount")

	ns := rcl.GetNamespace()

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    CommonObjectLabels(rcl),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: crGVK.Group,
			Kind:     crGVK.Kind,
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  saGVK.Group,
			Kind:      saGVK.Kind,
			Namespace: ns,
			Name:      owner.Name, // {kind}-adapter or {kind}-i-{name}
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

// TMCommonObjectLabels returns a set of labels which are always applied to
// TriggerMesh objects reconciled for the given component type.
func TMCommonObjectLabels(o kmeta.OwnerRefable) labels.Set {
	return labels.Set{
		appNameLabel:      ComponentName(o),
		appComponentLabel: componentAdapter,
		appPartOfLabel:    partOf,
		appManagedByLabel: managedBy,
	}
}

// CommonObjectLabels set of labels which are always applied to
// resource objects reconciled for the given component type.
var CommonObjectLabels = TMCommonObjectLabels

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

	for k, v := range overrides.NodeSelector {
		opts = append(opts, resource.NodeSelector(map[string]string{k: v}))
	}

	if overrides.Affinity != nil {
		opts = append(opts, resource.Affinity(*overrides.Affinity))
	}

	for _, t := range overrides.Env {
		opts = append(opts, resource.EnvVar(t.Name, t.Value))
	}

	for k, v := range overrides.Labels {
		opts = append(opts, resource.Label(k, v))
		opts = append(opts, resource.PodLabel(k, v))
	}
	for k, v := range overrides.Annotations {
		opts = append(opts, resource.Annotation(k, v))
		opts = append(opts, resource.PodAnnotation(k, v))
	}
	return opts
}

// toQuantity converts corev1.ResourceList to separate CPU and Memory quantities.
func toQuantity(resources corev1.ResourceList) (cpu *kres.Quantity, mem *kres.Quantity) {
	for k, v := range resources {
		switch k {
		case corev1.ResourceCPU:
			v := v
			cpu = &v
		case corev1.ResourceMemory:
			v := v
			mem = &v
		}
	}

	return
}
