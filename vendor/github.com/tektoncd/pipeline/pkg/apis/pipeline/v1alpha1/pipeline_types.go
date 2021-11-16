/*
Copyright 2019 The Tekton Authors

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

package v1alpha1

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipeline/dag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineSpec defines the desired state of Pipeline.
type PipelineSpec struct {
	// Description is a user-facing description of the pipeline that may be
	// used to populate a UI.
	// +optional
	Description string `json:"description,omitempty"`
	// Resources declares the names and types of the resources given to the
	// Pipeline's tasks as inputs and outputs.
	Resources []PipelineDeclaredResource `json:"resources,omitempty"`
	// Tasks declares the graph of Tasks that execute when this Pipeline is run.
	Tasks []PipelineTask `json:"tasks,omitempty"`
	// Params declares a list of input parameters that must be supplied when
	// this Pipeline is run.
	Params []ParamSpec `json:"params,omitempty"`
	// Workspaces declares a set of named workspaces that are expected to be
	// provided by a PipelineRun.
	// +optional
	Workspaces []PipelineWorkspaceDeclaration `json:"workspaces,omitempty"`
	// Results are values that this pipeline can output once run
	// +optional
	Results []PipelineResult `json:"results,omitempty"`
}

// PipelineResult used to describe the results of a pipeline
type PipelineResult = v1beta1.PipelineResult

// Check that Pipeline may be validated and defaulted.

// TaskKind defines the type of Task used by the pipeline.
type TaskKind = v1beta1.TaskKind

const (
	// NamespacedTaskKind indicates that the task type has a namepace scope.
	NamespacedTaskKind TaskKind = v1beta1.NamespacedTaskKind
	// ClusterTaskKind indicates that task type has a cluster scope.
	ClusterTaskKind TaskKind = v1beta1.ClusterTaskKind
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:noStatus

// Pipeline describes a list of Tasks to execute. It expresses how outputs
// of tasks feed into inputs of subsequent tasks.
// +k8s:openapi-gen=true
type Pipeline struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Pipeline from the client
	// +optional
	Spec PipelineSpec `json:"spec"`

	// Status is deprecated.
	// It usually is used to communicate the observed state of the Pipeline from
	// the controller, but was unused as there is no controller for Pipeline.
	// +optional
	Status *PipelineStatus `json:"status,omitempty"`
}

// PipelineStatus does not contain anything because Pipelines on their own
// do not have a status, they just hold data which is later used by a
// PipelineRun.
// Deprecated
type PipelineStatus struct {
}

func (p *Pipeline) PipelineMetadata() metav1.ObjectMeta {
	return p.ObjectMeta
}

func (p *Pipeline) PipelineSpec() PipelineSpec {
	return p.Spec
}

func (p *Pipeline) Copy() PipelineObject {
	return p.DeepCopy()
}

// PipelineTask defines a task in a Pipeline, passing inputs from both
// Params and from the output of previous tasks.
type PipelineTask struct {
	// Name is the name of this task within the context of a Pipeline. Name is
	// used as a coordinate with the `from` and `runAfter` fields to establish
	// the execution order of tasks relative to one another.
	Name string `json:"name,omitempty"`

	// TaskRef is a reference to a task definition.
	// +optional
	TaskRef *TaskRef `json:"taskRef,omitempty"`

	// TaskSpec is specification of a task
	// +optional
	TaskSpec *TaskSpec `json:"taskSpec,omitempty"`

	// Conditions is a list of conditions that need to be true for the task to run
	// +optional
	Conditions []PipelineTaskCondition `json:"conditions,omitempty"`

	// Retries represents how many times this task should be retried in case of task failure: ConditionSucceeded set to False
	// +optional
	Retries int `json:"retries,omitempty"`

	// RunAfter is the list of PipelineTask names that should be executed before
	// this Task executes. (Used to force a specific ordering in graph execution.)
	// +optional
	RunAfter []string `json:"runAfter,omitempty"`

	// Resources declares the resources given to this task as inputs and
	// outputs.
	// +optional
	Resources *PipelineTaskResources `json:"resources,omitempty"`
	// Parameters declares parameters passed to this task.
	// +optional
	Params []Param `json:"params,omitempty"`

	// Workspaces maps workspaces from the pipeline spec to the workspaces
	// declared in the Task.
	// +optional
	Workspaces []WorkspacePipelineTaskBinding `json:"workspaces,omitempty"`

	// Time after which the TaskRun times out. Defaults to 1 hour.
	// Specified TaskRun timeout should be less than 24h.
	// Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

func (pt PipelineTask) HashKey() string {
	return pt.Name
}

func (pt PipelineTask) Deps() []string {
	deps := []string{}
	deps = append(deps, pt.RunAfter...)
	if pt.Resources != nil {
		for _, rd := range pt.Resources.Inputs {
			deps = append(deps, rd.From...)
		}
	}
	// Add any dependents from conditional resources.
	for _, cond := range pt.Conditions {
		for _, rd := range cond.Resources {
			deps = append(deps, rd.From...)
		}
		for _, param := range cond.Params {
			expressions, ok := v1beta1.GetVarSubstitutionExpressionsForParam(param)
			if ok {
				resultRefs := v1beta1.NewResultRefs(expressions)
				for _, resultRef := range resultRefs {
					deps = append(deps, resultRef.PipelineTask)
				}
			}
		}
	}
	// Add any dependents from task results
	for _, param := range pt.Params {
		expressions, ok := v1beta1.GetVarSubstitutionExpressionsForParam(param)
		if ok {
			resultRefs := v1beta1.NewResultRefs(expressions)
			for _, resultRef := range resultRefs {
				deps = append(deps, resultRef.PipelineTask)
			}
		}
	}

	return deps
}

type PipelineTaskList []PipelineTask

func (l PipelineTaskList) Items() []dag.Task {
	tasks := []dag.Task{}
	for _, t := range l {
		tasks = append(tasks, dag.Task(t))
	}
	return tasks
}

func (l PipelineTaskList) Deps() map[string][]string {
	deps := map[string][]string{}
	for _, pt := range l {
		deps[pt.HashKey()] = pt.Deps()
	}
	return deps
}

// PipelineTaskParam is used to provide arbitrary string parameters to a Task.
type PipelineTaskParam = v1beta1.PipelineTaskParam

// PipelineTaskCondition allows a PipelineTask to declare a Condition to be evaluated before
// the Task is run.
type PipelineTaskCondition = v1beta1.PipelineTaskCondition

// PipelineDeclaredResource is used by a Pipeline to declare the types of the
// PipelineResources that it will required to run and names which can be used to
// refer to these PipelineResources in PipelineTaskResourceBindings.
type PipelineDeclaredResource = v1beta1.PipelineDeclaredResource

// PipelineTaskResources allows a Pipeline to declare how its DeclaredPipelineResources
// should be provided to a Task as its inputs and outputs.
type PipelineTaskResources = v1beta1.PipelineTaskResources

// PipelineTaskInputResource maps the name of a declared PipelineResource input
// dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources
// that should be used. This input may come from a previous task.
type PipelineTaskInputResource = v1beta1.PipelineTaskInputResource

// PipelineTaskOutputResource maps the name of a declared PipelineResource output
// dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources
// that should be used.
type PipelineTaskOutputResource = v1beta1.PipelineTaskOutputResource

// TaskRef can be used to refer to a specific instance of a task.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type TaskRef = v1beta1.TaskRef

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}
