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

package tekton

import (
	"context"
	"io/ioutil"
	"net/url"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck
	tektoninject "github.com/tektoncd/pipeline/pkg/client/injection/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/pkg/injection"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	e2ece "github.com/triggermesh/triggermesh/test/e2e/framework/cloudevents"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
	"github.com/triggermesh/triggermesh/test/e2e/framework/tekton"
)

/*
 * This test suite requires Tekton to be installed on the cluster.
 */

var targetAPIVersion = schema.GroupVersion{
	Group:   "targets.triggermesh.io",
	Version: "v1alpha1",
}

var tektonAPIVersion = schema.GroupVersion{
	Group:   "tekton.dev",
	Version: "v1beta1",
}

const (
	targetKind       = "TektonTarget"
	targetResource   = "tektontargets"
	taskKind         = "Task"
	taskResource     = "tasks"
	pipelineKind     = "Pipeline"
	pipelineResource = "pipelines"
)

var _ = FDescribe("Tekton Target", func() {
	f := framework.New("tektontarget")
	var ns string
	var tgtClient dynamic.ResourceInterface
	var taskClient dynamic.ResourceInterface
	var pipelineClient dynamic.ResourceInterface

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := targetAPIVersion.WithResource(targetResource)
		tgtClient = f.DynamicClient.Resource(gvr).Namespace(ns)

		taskClient = f.DynamicClient.Resource(tektonAPIVersion.WithResource(taskResource)).Namespace(ns)
		pipelineClient = f.DynamicClient.Resource(tektonAPIVersion.WithResource(pipelineResource)).Namespace(ns)
	})

	Context("a target is deployed", func() {
		var err error
		var task, pipeline *unstructured.Unstructured

		BeforeEach(func() {
			By("creating a Tekton Task", func() {
				task, err = createTask(taskClient, ns, "test-task-")
				Expect(err).ToNot(HaveOccurred())
			})

			By("creating a Tekton Pipeline", func() {
				pipeline, err = createPipeline(pipelineClient, ns, "test-pipeline-", task.GetName())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("the spec contains the default settings", func() {
			var tgtURL *url.URL

			BeforeEach(func() {
				By("creating the Tekton Target object", func() {
					tgt, err := createTarget(tgtClient, ns, "test-")
					Expect(err).ToNot(HaveOccurred())

					tgt = ducktypes.WaitUntilReady(f.DynamicClient, tgt)
					tgtURL = ducktypes.Address(tgt)
					Expect(tgtURL).ToNot(BeNil())
				})
			})

			When("a Tekton task event is sent to the target", func() {
				BeforeEach(func() {
					By("sending an event", func() {
						ev, err := createTektonEvent("task", task.GetName(), map[string]string{"greeting": "e2etest"}, f)
						Expect(err).ToNot(HaveOccurred())

						job := e2ece.RunEventSender(f.KubeClient, ns, tgtURL.String(), ev)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("creates the task", func() {
					tektonContext, _ := injection.Default.SetupInformers(context.Background(), f.ClientConfig())

					By("creating a taskrun object", func() {
						tektonClient := tektoninject.Get(tektonContext)
						tekton.WaitForTaskRunCompletion(tektonClient, ns, task.GetName()+"-12345")
					})

					By("verifying the build results", func() {
						log := tekton.GetPodLogs(f.KubeClient, ns, task.GetName()+"-12345-pod")
						bs, err := ioutil.ReadAll(log)
						Expect(err).ToNot(HaveOccurred())
						Expect(string(bs)).To(Equal("Hello world e2etest\n"))
					})
				})
			})

			When("a Tekton pipeline event is sent to the target", func() {
				BeforeEach(func() {
					By("sending an event", func() {
						ev, err := createTektonEvent("pipeline", pipeline.GetName(), nil, f)
						Expect(err).ToNot(HaveOccurred())

						job := e2ece.RunEventSender(f.KubeClient, ns, tgtURL.String(), ev)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("creates the pipeline", func() {
					tektonContext, _ := injection.Default.SetupInformers(context.Background(), f.ClientConfig())

					By("creating a pipelinerun object", func() {
						tektonClient := tektoninject.Get(tektonContext)
						tekton.WaitForPipelineRunCompletion(tektonClient, ns, pipeline.GetName()+"-12345")
					})

					By("verifying the build results", func() {
						log := tekton.GetPodLogs(f.KubeClient, ns, pipeline.GetName()+"-12345-greeting-pipeline-pod")
						bs, err := ioutil.ReadAll(log)
						Expect(err).ToNot(HaveOccurred())
						Expect(string(bs)).To(Equal("Hello world e2epipeline\n"))
					})
				})
			})
		})

		When("a client creates a target object with invalid specs", func() {
			Specify("the API server rejects the creation of that object", func() {
				By("setting an invalid success reap interval", func() {
					_, err := createTarget(tgtClient, ns, "test-invalid-success-reap-interval",
						withReapSuccessInterval("badinterval"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("is invalid: spec.reapPolicy.success: Invalid value: "))
				})

				By("setting an invalid fail reap interval", func() {
					_, err := createTarget(tgtClient, ns, "test-invalid-fail-reap-interval",
						withReapFailureInterval("badinterval"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("is invalid: spec.reapPolicy.fail: Invalid value: "))
				})
			})
		})
	})
})

// createTarget creates a TektonTarget object. The only tunable options affect the reaper, which won't matter here.
func createTarget(tgtClient dynamic.ResourceInterface, namespace, namePrefix string, opts ...targetOption) (*unstructured.Unstructured, error) {
	tgt := &unstructured.Unstructured{}
	tgt.SetAPIVersion(targetAPIVersion.String())
	tgt.SetKind(targetKind)
	tgt.SetNamespace(namespace)
	tgt.SetGenerateName(namePrefix)

	for _, opt := range opts {
		opt(tgt)
	}

	return tgtClient.Create(context.Background(), tgt, metav1.CreateOptions{})
}

// createTask creates a task object that will echo our greeting.
func createTask(taskClient dynamic.ResourceInterface, namespace, namePrefix string) (*unstructured.Unstructured, error) {
	task := &unstructured.Unstructured{}
	task.SetAPIVersion(tektonAPIVersion.String())
	task.SetKind(taskKind)
	task.SetNamespace(namespace)
	task.SetGenerateName(namePrefix)

	paramsArray := []interface{}{map[string]interface{}{
		"name":        "greeting",
		"type":        "string",
		"description": "Our greeting",
	}}

	stepsArray := []interface{}{map[string]interface{}{
		"args":    []interface{}{"Hello", "world", "$(params.greeting)"},
		"command": []interface{}{"echo"},
		"image":   "centos",
		"name":    "first-action",
	}}

	err := unstructured.SetNestedSlice(task.Object, paramsArray, "spec", "params")
	if err != nil {
		return nil, err
	}

	err = unstructured.SetNestedSlice(task.Object, stepsArray, "spec", "steps")
	if err != nil {
		return nil, err
	}

	return taskClient.Create(context.Background(), task, metav1.CreateOptions{})
}

// createPipeline creates a pipeline to invoke the recently created task
func createPipeline(pipelineClient dynamic.ResourceInterface, namespace, namePrefix, taskName string) (*unstructured.Unstructured, error) {
	pipeline := &unstructured.Unstructured{}
	pipeline.SetAPIVersion(tektonAPIVersion.String())
	pipeline.SetKind(pipelineKind)
	pipeline.SetNamespace(namespace)
	pipeline.SetGenerateName(namePrefix)

	pipelineArray := []interface{}{map[string]interface{}{
		"name": "greeting-pipeline",
		"taskRef": map[string]interface{}{
			"name": taskName,
		},
		"params": []interface{}{map[string]interface{}{
			"name":  "greeting",
			"value": "e2epipeline",
		}},
	}}

	err := unstructured.SetNestedSlice(pipeline.Object, pipelineArray, "spec", "tasks")
	if err != nil {
		return nil, err
	}

	return pipelineClient.Create(context.Background(), pipeline, metav1.CreateOptions{})
}

// createTektonEvent will create a CloudEvent to conform to the Tekton target. Note the params is not used when invoking a pipeline.
func createTektonEvent(buildType, name string, params map[string]string, f *framework.Framework) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetID("12345")
	event.SetType("io.triggermesh.tekton.run")
	event.SetSource("e2e.triggermesh")
	event.SetExtension("iotriggermeshe2e", f.UniqueName)

	data := map[string]interface{}{
		"buildType": buildType,
		"name":      name,
		"params":    params,
	}

	err := event.SetData(cloudevents.ApplicationJSON, data)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

type targetOption func(*unstructured.Unstructured)

func withReapSuccessInterval(interval string) targetOption {
	return func(tgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(tgt.Object, interval, "spec", "reapPolicy", "success"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.reapPolicy.success field: %s", err)
		}
	}
}

func withReapFailureInterval(interval string) targetOption {
	return func(tgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(tgt.Object, interval, "spec", "reapPolicy", "fail"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.reapPolicy.fail field: %s", err)
		}
	}
}
