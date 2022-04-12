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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const codeVersionAnnotation = "extensions.triggermesh.io/codeVersion"

const klrEntrypoint = "/opt/aws-custom-runtime"

const (
	eventStoreEnv    = "EVENTSTORE_URI"
	runtimeEnvPrefix = "RUNTIME_"
)

// adapterConfig contains properties used to configure the Function's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterServiceBuilder.
var _ common.AdapterServiceBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterServiceBuilder.
func (r *Reconciler) BuildAdapter(rcl commonv1alpha1.Reconcilable, sinkURI *apis.URL) *servingv1.Service {
	f := rcl.(*v1alpha1.Function)

	srcCodePath := filepath.Join("/opt", "source."+fileExtension(f.Spec.Runtime))

	var cmapName string
	var cmapRev string
	if codeCmap := f.Status.ConfigMap; codeCmap != nil {
		cmapName = codeCmap.Name
		cmapRev = codeCmap.ResourceVersion
	}

	handler := "source." + f.Spec.Entrypoint

	ceOverrides := map[string]string{
		// Default values for required attributes
		"type":   f.GetEventTypes()[0],
		"source": f.AsEventSource(),
	}

	if f.Spec.CloudEventOverrides != nil {
		for k, v := range f.Spec.CloudEventOverrides.Extensions {
			if k != "type" && k != "source" {
				ceOverrides[k] = v
			}
		}
	}

	var responseMode string
	if f.Spec.ResponseIsEvent {
		responseMode = "event"
	}

	ksvcVisibility := resource.VisibilityClusterLocal
	if f.Spec.Public {
		ksvcVisibility = resource.VisibilityPublic
	}

	return common.NewAdapterKnService(rcl, sinkURI,
		resource.Image(lookupRuntimeImage(f.Spec.Runtime)),

		ksvcVisibility,
		resource.Annotation(codeVersionAnnotation, cmapRev),

		resource.Label(functionNameLabel, f.Name),

		resource.EnvVar(eventStoreEnv, f.Spec.EventStore.URI),
		resource.EnvVar("_HANDLER", handler),
		resource.EnvVar("RESPONSE_FORMAT", "CLOUDEVENTS"),
		resource.EnvVar("CE_FUNCTION_RESPONSE_MODE", responseMode),
		resource.EnvVars(sortedEnvVarsWithPrefix("CE_OVERRIDES_", ceOverrides)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),

		resource.ConfigMapMount("code", srcCodePath, cmapName, codeCmapDataKey),

		resource.EntrypointCommand(klrEntrypoint),
	)
}

// RBACOwners implements common.AdapterServiceBuilder.
func (r *Reconciler) RBACOwners(fn commonv1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error) {
	trgs, err := r.fnLister(fn.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(trgs))
	for i := range trgs {
		ownerRefables[i] = trgs[i]
	}

	return ownerRefables, nil
}

// Lambda runtimes require file extensions to match the language,
// i.e. source file for Python runtime must have ".py" prefix, JavaScript - ".js", etc.
// It would be more correct to declare these extensions explicitly,
// along with the runtime container URIs, but since we manage the
// available runtimes list, this also works.
func fileExtension(runtime string) string {
	runtime = strings.ToLower(runtime)
	switch {
	case strings.Contains(runtime, "python"):
		return "py"
	case strings.Contains(runtime, "node") ||
		strings.Contains(runtime, "js"):
		return "js"
	case strings.Contains(runtime, "ruby"):
		return "rb"
	case strings.Contains(runtime, "sh"):
		return "sh"
	}
	return "txt"
}

// Env variables from extensions override map are sorted alphabetically before
// passing to container env to prevent reconciliation loop when map keys are randomized.
func sortedEnvVarsWithPrefix(prefix string, overrides map[string]string) []corev1.EnvVar {
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	res := make([]corev1.EnvVar, 0, len(keys))
	for _, key := range keys {
		res = append(res, corev1.EnvVar{
			Name:  strings.ToUpper(prefix + key),
			Value: overrides[key],
		})
	}

	return res
}

var (
	// guards initialization by initRuntimes, which populates runtimes
	runtimesOnce sync.Once
	// runtime names and associated container images
	runtimes map[string]string
)

func initRuntimes() {
	runtimes = make(map[string]string)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, runtimeEnvPrefix) {
			continue
		}
		e = strings.TrimPrefix(e, runtimeEnvPrefix)
		runtimePairs := strings.SplitN(e, "=", 2)
		runtimes[runtimePairs[0]] = runtimePairs[1]
	}
}

func lookupRuntimeImage(runtime string) string {
	rn := strings.ToLower(runtime)

	runtimesOnce.Do(initRuntimes)

	for name, img := range runtimes {
		name = strings.ToLower(name)
		if strings.Contains(name, rn) {
			return img
		}
	}

	return ""
}
