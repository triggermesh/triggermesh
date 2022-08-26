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
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"

	"github.com/triggermesh/triggermesh/pkg/apis/flow"
	"github.com/triggermesh/triggermesh/pkg/metrics"
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
	sink     string

	mt *pkgadapter.MetricTag
	sr *metrics.EventProcessingStatsReporter
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: flow.DataWeaveTransformationResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

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
		sink:     env.Sink,
		mt:       mt,
		sr:       metrics.MustNewEventProcessingStatsReporter(mt),
	}

	if env.DwSpell != "" {
		adapter.defaultSpell = &env.DwSpell
	}
	if env.InputContentType != "" {
		adapter.defaultInputContentType = &env.InputContentType
	}
	if env.OutputContentType != "" {
		adapter.defaultOutputContentType = &env.OutputContentType
	} else {
		adapter.defaultOutputContentType = ptr.String("application/json")
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
	var sout bytes.Buffer
	var serr bytes.Buffer

	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	err = validateContentType(event.DataContentType(), event.Data())
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New(err.Error()), nil)
	}

	inputData := event.Data()
	spell := a.defaultSpell
	inputContentType := a.defaultInputContentType
	outputContentType := a.defaultOutputContentType

	req := &DataWeaveTransformationStructuredRequest{}
	if err := event.DataAs(req); err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if !a.spellOverride && req.Spell != "" {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("it is not allowed to override Spell per CloudEvent"), nil)
	}

	if req.InputContentType != "" || req.OutputContentType != "" || req.Spell != "" {
		if req.InputData == "" {
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
				errors.New("inputData not found"), nil)
		}
	}

	if a.spellOverride {
		if req.InputContentType != "" {
			inputContentType = &req.InputContentType
		}

		if req.OutputContentType != "" {
			outputContentType = &req.OutputContentType
		}

		if req.InputData != "" {
			inputData = []byte(req.InputData)
		}

		if req.Spell != "" {
			spell = &req.Spell
		}
	}

	if inputContentType == nil || outputContentType == nil || inputData == nil || spell == nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("parameters not found"), nil)
	}

	err = validateContentType(*inputContentType, inputData)
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New(err.Error()), nil)
	}

	errs := registerAndPopulateSpell(*spell, dwFolder)
	if errs != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the spell")
	}

	cmd := exec.Command("dw", "--local-spell", dwFolder)
	cmd.Env = append(cmd.Env, "DW_HOME="+path)
	cmd.Env = append(cmd.Env, "DW_DEFAULT_INPUT_MIMETYPE="+*inputContentType)

	cmd.Stdout = &sout
	cmd.Stdin = bytes.NewReader([]byte(inputData))
	cmd.Stderr = &serr

	// Add a cleanup method in case of success/failure. We're not interested in
	// the failure scenario here as it could fail with the directory not existing
	defer func() {
		_ = os.RemoveAll(dwFolder)
	}()

	err = cmd.Run()
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "executing the spell")
	}

	cleaned := bytes.Trim(sout.Bytes(), "Running local spell")
	if err := event.SetData(*outputContentType, cleaned); err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	event.SetType(event.Type() + ".response")
	a.logger.Debugf("Responding with transformed event: %s", event.Type())
	if a.sink != "" {
		if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			a.logger.Errorw("Error sending event to sink", zap.Error(result))
		}
		a.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
		return nil, cloudevents.ResultACK
	}

	a.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	return &event, cloudevents.ResultACK
}

// registerAndPopulateSpell create the DataWeave spell and populate.
func registerAndPopulateSpell(spell, dwFolder string) error {
	if err := exec.Command("dw", "--new-spell", dwFolder).Run(); err != nil {
		return err
	}

	if err := os.WriteFile(dwFolder+"/src/Main.dwl", []byte(spell), 0644); err != nil {
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
