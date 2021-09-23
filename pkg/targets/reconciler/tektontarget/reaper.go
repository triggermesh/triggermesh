/*
Copyright (c) 2021 TriggerMesh Inc.

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

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/hashicorp/go-uuid"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "knative.dev/pkg/client/injection/kube/client"

	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// reaperThread Run at a set interval to trigger each namespace's reaping functionality
func reaperThread(ctx context.Context, r *reconciler) {
	interval, _ := time.ParseDuration(r.adapterCfg.ReapingInterval)
	poll := time.NewTicker(interval)
	log := logging.FromContext(ctx)

	p, err := cloudevents.NewHTTP()
	if err != nil {
		log.Fatal("failed to create new HTTP based cloudevent: ", zap.Error(err))
	}

	client, err := cloudevents.NewClient(p)
	if err != nil {
		log.Fatal("Unable to create cloudevent client: ", zap.Error(err))
	}

	for {
		<-poll.C // Used to wait for the poll timer
		log.Debug("executing reaping")
		nsl, err := k8sclient.Get(ctx).CoreV1().Namespaces().List(ctx, v1.ListOptions{})
		if err != nil {
			log.Fatal("Unable to retrieve namespaces: ", zap.Error(err))
		}

		// search for tektontargets across all namespaces
		for _, ns := range nsl.Items {
			targets, err := r.targetLister(ns.Name).List(labels.Everything())
			if err != nil {
				log.Warn("unable to retrieve targets for #{ns.Name}: ", zap.Error(err))
				continue
			}

			for _, t := range targets {
				// Abort if the target isn't ready
				if !t.Status.IsReady() || t.Status.Address == nil ||
					t.Status.Address.URL == nil || t.Status.Address.URL.IsEmpty() {
					continue
				}

				log.Info("Found target: " + t.Namespace + "." + t.Name)
				// Send the reap cloudevent
				cloudCtx := cloudevents.ContextWithTarget(ctx, t.Status.Address.URL.String())

				id, _ := uuid.GenerateUUID()

				newEvent := cloudevents.NewEvent(cloudevents.VersionV1)
				newEvent.SetType(v1alpha1.EventTypeTektonReap)
				newEvent.SetSource("CronJob")
				newEvent.SetTime(time.Now())
				newEvent.SetID(id)

				// This should always pass
				_ = newEvent.SetData(cloudevents.ApplicationJSON, nil)

				result := client.Send(cloudCtx, newEvent)
				log.Info("event sending results: ", zap.Error(result))
			}
		}
	}
}
