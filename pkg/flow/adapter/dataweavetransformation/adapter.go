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

package dataweavetransformation

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/flow"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

const (
	path     = "/tmp/dw"
	dwFolder = path + "/custom"
)

var _ pkgadapter.Adapter = (*dataweaveTransformAdapter)(nil)

type dataweaveTransformAdapter struct {
	defaultSpell             *string
	defaultInputContentType  *string
	defaultOutputContentType *string
	spellOverride            bool

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag
	sink     string
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: flow.DataWeaveTransformationResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envAccessor)

	if err := env.validate(); err != nil {
		logger.Panicf("Configuration error: %v", err)
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticDataContentType(cloudevents.ApplicationXML),
		targetce.ReplierWithStaticErrorDataContentType(*cloudevents.StringOfApplicationJSON()),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(targetce.PayloadPolicyAlways)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	adapter := &dataweaveTransformAdapter{
		spellOverride: env.AllowDwSpellOverride,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
		mt:       mt,
		sink:     env.Sink,
	}

	if env.DwSpell != "" {
		adapter.defaultSpell = &env.DwSpell
	}
	if env.InputContentType != "" {
		adapter.defaultInputContentType = &env.InputContentType
	}
	if env.OutputContentType != "" {
		adapter.defaultOutputContentType = &env.OutputContentType
	}

	return adapter
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *dataweaveTransformAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting DataWeave transformer")
	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *dataweaveTransformAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var err error
	var tmpFile *os.File
	var inputData []byte
	var spell string
	var inputContentType string
	var outputContentType string

	err = validateContentType(event.DataContentType(), event.Data())
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New(err.Error()), nil)
	}

	req := &DataWeaveTransformationStructuredRequest{}
	if err := event.DataAs(req); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if !a.spellOverride && req.Spell != "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("it is not allowed to override Spell per CloudEvent"), nil)
	}

	if a.defaultSpell == nil && req.Spell == "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("no default Spell or in request found"), nil)
	}

	if req.InputContentType != "" || req.OutputContentType != "" || req.Spell != "" {
		if req.InputData == "" {
			return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
				errors.New("inputData not found"), nil)
		}
	}

	// Check for spellOverride to be enabled and all the parameters to be present.
	if a.spellOverride && req.InputData != "" && req.InputContentType != "" && req.OutputContentType != "" && req.Spell != "" {
		inputData = []byte(req.InputData)
		spell = req.Spell
		inputContentType = req.InputContentType
		outputContentType = req.OutputContentType

		// In the case that it receives the inputData in a dict instead of directly,
		// it will use the default parameters present in the yaml and the inputData.
	} else {
		if req.InputData != "" {
			inputData = []byte(req.InputData)
		} else {
			inputData = event.Data()
		}
		if a.defaultSpell != nil && a.defaultInputContentType != nil && a.defaultOutputContentType != nil {
			spell = *a.defaultSpell
			inputContentType = *a.defaultInputContentType
			outputContentType = *a.defaultOutputContentType
		}
	}

	err = validateContentType(inputContentType, inputData)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New(err.Error()), nil)
	}

	switch inputContentType {
	case "application/json":
		tmpFile, err = ioutil.TempFile(path, "*.json")
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the json file")
		}

	case "application/xml":
		tmpFile, err = ioutil.TempFile(path, "*.xml")
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the xml file")
		}
	}

	errs := registerAndPopulateSpell(spell, dwFolder)
	if errs != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the spell")
	}

	if _, err := tmpFile.Write(inputData); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "writing to the file")
	}

	out, err := exec.Command("dw", "-i", "payload", tmpFile.Name(), "--local-spell", dwFolder).Output()
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "executing the spell")
	}

	err = os.Remove(tmpFile.Name())
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "removing the file")
	}

	err = os.RemoveAll(dwFolder)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "removing the folder")
	}

	cleaned := bytes.Trim(out, "Running local spell")
	if err := event.SetData(outputContentType, cleaned); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	event.SetType(event.Type() + ".response")
	a.logger.Infof("responding with transformed event: %v", event.Type())
	if a.sink != "" {
		if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
			a.logger.Errorf("Error sending event to sink: %v", result)
		}
		return nil, cloudevents.ResultACK
	}

	return &event, cloudevents.ResultACK
}

// registerAndPopulateSpell create the DataWeave spell and populate.
func registerAndPopulateSpell(spell, dwFolder string) error {
	if err := exec.Command("dw", "--new-spell", dwFolder).Run(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(dwFolder+"/src/Main.dwl", []byte(spell), 0644); err != nil {
		return err
	}

	return nil
}

func isValidXML(data []byte) bool {
	return xml.Unmarshal(data, new(interface{})) == nil
}

func isValidJSON(data []byte) bool {
	return json.Unmarshal(data, new(interface{})) == nil
}

func validateContentType(contentType string, data []byte) error {
	switch contentType {
	case "application/json":
		if !isValidJSON(data) {
			return errors.New("invalid Json")
		}
	case "application/xml":
		if !isValidXML(data) {
			return errors.New("invalid XML")
		}
	default:
		return errors.New("unexpected type for the incoming event")
	}
	return nil
}
