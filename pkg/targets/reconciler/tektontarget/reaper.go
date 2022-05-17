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
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// reaperThread Run at a set interval to trigger each namespace's reaping functionality
func reaperThread(ctx context.Context, r *Reconciler) {
	interval, _ := time.ParseDuration(r.adapterCfg.ReapingInterval)
	poll := time.NewTicker(interval)
	log := logging.FromContext(ctx)

	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalw("Unable to create CloudEvent client", zap.Error(err))
	}

	for {
		<-poll.C // Used to wait for the poll timer
		log.Debug("Executing reaping")

		targets, err := r.trgLister.List(labels.Everything())
		if err != nil {
			log.Errorw("Unable to list TektonTargets from cache", zap.Error(err))
			continue
		}

		for _, t := range targets {
			// Abort if the target isn't ready
			if !t.Status.GetCondition(apis.ConditionReady).IsTrue() ||
				t.Status.Address == nil || t.Status.Address.URL.IsEmpty() {
				continue
			}

			log.Info("Found target: ", t.Namespace+"."+t.Name)

			// Send the reap CloudEvent
			cloudCtx := cloudevents.ContextWithTarget(ctx, t.Status.Address.URL.String())

			newEvent := cloudevents.NewEvent(cloudevents.VersionV1)
			newEvent.SetType(v1alpha1.EventTypeTektonReap)
			newEvent.SetSource("triggermesh-controller")

			if err := newEvent.SetData(cloudevents.ApplicationJSON, nil); err != nil {
				log.Errorw("Failed to set event data", zap.Error(err))
				continue
			}

			if result := client.Send(cloudCtx, newEvent); !cloudevents.IsACK(result) {
				log.Errorw("Event wasn't acknowledged", zap.Error(result))
			}
		}
	}
}
