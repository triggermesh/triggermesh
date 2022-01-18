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

package transformation

import (
	"fmt"
	"log"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer/add"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer/delete"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer/shift"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer/store"
)

// Pipeline is a set of Transformations that are
// sequentially applied to JSON data.
type Pipeline struct {
	Transformers []transformer.Transformer
}

// register loads available Transformation into a named map.
func register() map[string]transformer.Transformer {
	transformations := make(map[string]transformer.Transformer)

	add.Register(transformations)
	delete.Register(transformations)
	shift.Register(transformations)
	store.Register(transformations)

	return transformations
}

// newPipeline loads available Transformations and creates a Pipeline.
func newPipeline(transformations []v1alpha1.Transform) (*Pipeline, error) {
	availableTransformers := register()
	pipeline := []transformer.Transformer{}

	for _, transformation := range transformations {
		operation, exist := availableTransformers[transformation.Operation]
		if !exist {
			return nil, fmt.Errorf("transformation %q not found", transformation.Operation)
		}
		for _, kv := range transformation.Paths {
			pipeline = append(pipeline, operation.New(kv.Key, kv.Value))
			log.Printf("%s: %s", transformation.Operation, kv.Key)
		}
	}

	return &Pipeline{
		Transformers: pipeline,
	}, nil
}

// SetStorage injects shared storage with Pipeline vars.
func (p *Pipeline) setStorage(s *storage.Storage) {
	for _, v := range p.Transformers {
		v.SetStorage(s)
	}
}

// InitStep runs Transformations that are marked as InitStep.
func (p *Pipeline) initStep(data []byte) {
	for _, v := range p.Transformers {
		if !v.InitStep() {
			continue
		}
		if _, err := v.Apply(data); err != nil {
			log.Printf("Failed to apply Init step: %v", err)
		}
	}
}

// Apply applies Pipeline transformations.
func (p *Pipeline) apply(data []byte) ([]byte, error) {
	var err error
	for _, v := range p.Transformers {
		if v.InitStep() {
			continue
		}
		data, err = v.Apply(data)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}
