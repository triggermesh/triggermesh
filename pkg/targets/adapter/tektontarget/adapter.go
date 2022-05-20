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

package tektontarget

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektoninject "github.com/tektoncd/pipeline/pkg/client/injection/client"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// Expected CloudEvent message reflecting the type of action to perform
type tektonMsg struct {
	BuildType string            `json:"buildtype"`
	Name      string            `json:"name"`
	Params    map[string]string `json:"params,omitempty"`
}

const (
	tektonTargetLabel = "name.tekton.targets.triggermesh.io"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.TektonTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	var successAge *time.Duration
	var failAge *time.Duration

	if env.ReapSuccessAge != "" {
		success, err := time.ParseDuration(env.ReapSuccessAge)
		if err != nil {
			logger.Fatal("error: unable to parse success duration: ", zap.Error(err))
		}
		successAge = &success
	}

	if env.ReapFailAge != "" {
		fail, err := time.ParseDuration(env.ReapFailAge)
		if err != nil {
			logger.Fatal("error: unable to parse failure duration: ", zap.Error(err))
		}
		failAge = &fail
	}

	return &tektonAdapter{
		tektonClient:   tektoninject.Get(ctx),
		ceClient:       ceClient,
		namespace:      envAcc.GetNamespace(),
		targetName:     envAcc.GetName(),
		reapSuccessAge: successAge,
		reapFailAge:    failAge,
		logger:         logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*tektonAdapter)(nil)

type tektonAdapter struct {
	namespace  string
	targetName string

	reapSuccessAge *time.Duration
	reapFailAge    *time.Duration

	tektonClient tektonclient.Interface
	ceClient     cloudevents.Client
	logger       *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (t *tektonAdapter) Start(ctx context.Context) error {
	t.logger.Info("Starting Tekton adapter")

	if err := t.ceClient.StartReceiver(ctx, t.dispatch); err != nil {
		return err
	}
	return nil
}

func (t *tektonAdapter) dispatch(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	typ := event.Type()

	if typ == v1alpha1.EventTypeTektonReap {
		return t.reapRuns(ctx)
	} else if typ != v1alpha1.EventTypeTektonRun {
		return fmt.Errorf("cannot process event with type %q", typ)
	}

	// Take a CloudEvent as passed in, and submit a taskrun or pipelinerun job
	msg := &tektonMsg{}
	if err := event.DataAs(msg); err != nil {
		return fmt.Errorf("error processing incoming event data: %w", err)
	}

	switch msg.BuildType {
	case "task":
		return t.submitTaskRun(ctx, msg, event.ID())
	case "pipeline":
		return t.submitPipelineRun(ctx, msg, event.ID())
	default:
		return fmt.Errorf("unknown build type %q", msg.BuildType)
	}
}

func (t *tektonAdapter) submitPipelineRun(ctx context.Context, msg *tektonMsg, id string) cloudevents.Result {
	var pipelineRun tektonapi.PipelineRun
	pipelineRun.SetName(msg.Name + "-" + id)
	pipelineRun.SetLabels(t.generateTargetLabel())

	pipelineRun.Spec.PipelineRef = &tektonapi.PipelineRef{
		Name: msg.Name,
	}

	if msg.Params != nil {
		pipelineRun.Spec.Params = generateParam(msg.Params)
	}

	runJob, err := t.tektonClient.TektonV1beta1().PipelineRuns(t.namespace).Create(ctx, &pipelineRun, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error generating pipeline run job: %w", err)
	}

	t.logger.Infof("Pipeline submitted as: %+v", runJob)
	return cloudevents.ResultACK
}

func (t *tektonAdapter) submitTaskRun(ctx context.Context, msg *tektonMsg, id string) cloudevents.Result {
	var taskRun tektonapi.TaskRun
	taskRun.SetName(msg.Name + "-" + id)
	taskRun.SetLabels(t.generateTargetLabel())

	taskRun.Spec.TaskRef = &tektonapi.TaskRef{
		Name: msg.Name,
	}

	if msg.Params != nil {
		taskRun.Spec.Params = generateParam(msg.Params)
	}

	runJob, err := t.tektonClient.TektonV1beta1().TaskRuns(t.namespace).Create(ctx, &taskRun, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error generating task run job: %w", err)
	}

	t.logger.Infof("Task submitted as: %+v", runJob)
	return cloudevents.ResultACK
}

func (t *tektonAdapter) reapRuns(ctx context.Context) cloudevents.Result {
	var err error

	// Verify if reaping should be allowed to happen
	if t.reapSuccessAge == nil && t.reapFailAge == nil {
		t.logger.Debug("No reaping interval specified")
		return cloudevents.ResultACK
	}

	taskList, err := t.tektonClient.TektonV1beta1().TaskRuns(t.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: tektonTargetLabel + "=" + t.targetName,
	})
	if err != nil {
		return fmt.Errorf("error retrieving list of jobs: %w", err)
	}

	pipelineList, err := t.tektonClient.TektonV1beta1().PipelineRuns(t.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: tektonTargetLabel + "=" + t.targetName,
	})
	if err != nil {
		return fmt.Errorf("error retrieving list of jobs: %w", err)
	}

	if taskList != nil {
		for _, v := range taskList.Items {
			// Skip jobs that are still running
			if v.Status.CompletionTime == nil {
				continue
			}

			status := v.Status.Conditions[0]
			// Skip jobs that have finished, but are still inside the reaping interval
			if status.Status == corev1.ConditionTrue && t.reapSuccessAge != nil {
				if v.Status.CompletionTime.Add(*t.reapSuccessAge).After(time.Now()) {
					continue
				}
			} else if t.reapFailAge != nil {
				if v.Status.CompletionTime.Add(*t.reapFailAge).After(time.Now()) {
					continue
				}
			}

			t.logger.Info("Reaping taskrun: ", v.Name)
			if err := t.tektonClient.TektonV1beta1().TaskRuns(t.namespace).Delete(ctx, v.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("error unable to delete task run object %q: %w", v.Name, err)
			}
		}
	}

	if pipelineList != nil {
		for _, v := range pipelineList.Items {
			// Skip jobs that are still running
			if v.Status.CompletionTime == nil {
				continue
			}

			status := v.Status.Conditions[0]
			// Skip jobs that have finished, but are still inside the reaping interval
			if status.Status == corev1.ConditionTrue && t.reapSuccessAge != nil {
				if v.Status.CompletionTime.Add(*t.reapSuccessAge).After(time.Now()) {
					continue
				}
			} else if t.reapFailAge != nil {
				if v.Status.CompletionTime.Add(*t.reapFailAge).After(time.Now()) {
					continue
				}
			}

			t.logger.Info("Reaping pipelinerun: ", v.Name)
			if err := t.tektonClient.TektonV1beta1().PipelineRuns(t.namespace).Delete(ctx, v.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("error unable to delete pipeline run object %q: %w", v.Name, err)
			}
		}
	}

	return cloudevents.ResultACK
}

func generateParam(params map[string]string) []tektonapi.Param {
	tektonParm := make([]tektonapi.Param, 0)

	for k, v := range params {
		p := tektonapi.Param{
			Name:  k,
			Value: tektonapi.ArrayOrString{Type: tektonapi.ParamTypeString, StringVal: v},
		}
		tektonParm = append(tektonParm, p)
	}

	return tektonParm
}

func (t *tektonAdapter) generateTargetLabel() map[string]string {
	labels := make(map[string]string)

	labels[tektonTargetLabel] = t.targetName

	return labels
}
