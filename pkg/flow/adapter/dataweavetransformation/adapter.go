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

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

var _ pkgadapter.Adapter = (*dataweaveTransformAdapter)(nil)

type dataweaveTransformAdapter struct {
	spell               string
	incomingContentType string
	outputContentType   string

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	sink     string
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticDataContentType(cloudevents.ApplicationXML),
		targetce.ReplierWithStaticErrorDataContentType(*cloudevents.StringOfApplicationJSON()),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(targetce.PayloadPolicyAlways)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	adapter := &dataweaveTransformAdapter{
		spell:               env.DwSpell,
		incomingContentType: env.IncomingContentType,
		outputContentType:   env.OutputContentType,
		replier:             replier,
		ceClient:            ceClient,
		logger:              logger,
		sink:                env.Sink,
	}

	return adapter
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *dataweaveTransformAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting DataWeave transformer")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *dataweaveTransformAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var err error
	var tmpfile *os.File
	path := "/opt/dw"
	dwFolder := path + "/custom"

	switch a.incomingContentType {
	case "application/json":
		if !isValidJSON(event.Data()) {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing,
				errors.New("invalid Json"), nil)
		}
		tmpfile, err = ioutil.TempFile(path, "*.json")
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the json file")
		}
	case "application/xml":
		if !isValidXML(event.Data()) {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing,
				errors.New("invalid XML"), nil)
		}
		tmpfile, err = ioutil.TempFile(path, "*.xml")
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the xml file")
		}
	default:
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing,
			errors.New("unexpected type for the incoming event"), nil)
	}

	errs := registerAndPopulateSpell(a.spell, dwFolder)
	if errs != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the spell")
	}

	if _, err := tmpfile.Write(event.Data()); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "writing to the file")
	}

	out, err := exec.Command("dw", "-i", "payload", tmpfile.Name(), "--local-spell", dwFolder).Output()
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "executing the spell")
	}

	err = os.Remove(tmpfile.Name())
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "removing the file")
	}

	err = os.RemoveAll(dwFolder)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "removing the folder")
	}

	cleaned := bytes.Trim(out, "Running local spell")
	if err := event.SetData(a.outputContentType, cleaned); err != nil {
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
