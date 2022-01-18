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

package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/kelseyhightower/envconfig"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation"
)

type envConfig struct {
	// Sink URL where to send cloudevents
	Sink string `envconfig:"K_SINK"`

	// Transformation specifications
	TransformationContext string `envconfig:"TRANSFORMATION_CONTEXT"`
	TransformationData    string `envconfig:"TRANSFORMATION_DATA"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Failed to process env var: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trnContext, trnData := []v1alpha1.Transform{}, []v1alpha1.Transform{}
	err := json.Unmarshal([]byte(env.TransformationContext), &trnContext)
	if err != nil {
		log.Fatalf("Cannot unmarshal Context Transformation variable: %v", err)
	}
	err = json.Unmarshal([]byte(env.TransformationData), &trnData)
	if err != nil {
		log.Fatalf("Cannot unmarshal Data Transformation variable: %v", err)
	}

	handler, err := transformation.NewHandler(trnContext, trnData)
	if err != nil {
		log.Fatalf("Cannot create transformation handler: %v", err)
	}

	if err := handler.Start(ctx, env.Sink); err != nil {
		log.Fatalf("Transformation handler: %v", err)
	}
}
