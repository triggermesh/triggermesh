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

package status

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// PropagateCondition propagates a status condition to the status of the given
// source object using the provided Patcher.
func PropagateCondition(ctx context.Context, p *Patcher, src *v1alpha1.AWSSNSSource, cond *apis.Condition) error {
	srcCpy := shallowSourceCopy(src)
	stMan := srcCpy.GetStatusManager()
	condMan := stMan.Manage(stMan)

	switch cond.Status {
	case corev1.ConditionTrue:
		condMan.MarkTrue(cond.Type)
	case corev1.ConditionFalse:
		condMan.MarkFalse(cond.Type, cond.Reason, cond.Message)
	}

	if c := ClockFromContext(ctx); c != nil {
		forceTransitionTime(stMan.GetConditions(), cond.Type, c)
	}

	patch, err := duck.CreatePatch(src, srcCpy)
	if err != nil {
		return fmt.Errorf("creating JSON patch for status condition: %w", err)
	}
	if len(patch) == 0 {
		return nil
	}

	if _, err = p.Patch(ctx, src.Name, patch); err != nil {
		return fmt.Errorf("applying JSON patch: %w", err)
	}
	return nil
}

// shallowSourceCopy returns a shallow copy of the provided source object, with
// the exception of Status.Conditions which are deeply copied. This allows
// applying modifications to those status conditions, then generating a JSON
// patch by comparing the before/after states.
func shallowSourceCopy(src *v1alpha1.AWSSNSSource) *v1alpha1.AWSSNSSource {
	srcCpy := *src
	srcCpy.Status.Conditions = src.Status.Conditions.DeepCopy()
	return &srcCpy
}

// forceTransitionTime forces the value of the status condition of the given
// type to the value returned by the provided Clock.
// This helper exists because Knative's SetConditon method always uses
// time.Now(), which is not suitable to write reproducible tests in some
// situations.
func forceTransitionTime(conds apis.Conditions, ct apis.ConditionType, c Clock) {
	for i, cond := range conds {
		if cond.Type == ct {
			conds[i].LastTransitionTime = c.Now()
		}
	}
}
