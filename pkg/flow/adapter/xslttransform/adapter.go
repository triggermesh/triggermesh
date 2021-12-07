/*
Copyright 2021 TriggerMesh Inc.

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

package xslttransform

import (
	"context"
	"errors"
	"fmt"

	"github.com/jbowtie/gokogiri/xml"
	"github.com/jbowtie/ratago/xslt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

var _ pkgadapter.Adapter = (*xsltTransformAdapter)(nil)

type xsltTransformAdapter struct {
	defaultXSLT  *xslt.Stylesheet
	xsltOverride bool

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

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

	adapter := &xsltTransformAdapter{
		xsltOverride: env.AllowXsltOverride,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
	}

	if env.Xslt != "" {
		style, err := parseXSLT(env.Xslt)
		if err != nil {
			logger.Panicf("Non valid XSLT document: %v", err)
		}

		adapter.defaultXSLT = style
	}

	return adapter
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *xsltTransformAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting XSLT transformer")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *xsltTransformAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	isStructuredTransform := event.Type() == v1alpha1.EventTypeXSLTTransform
	if isStructuredTransform && !a.xsltOverride {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("it is not allowed to override XSLT per CloudEvent"), nil)
	}

	isXML := event.DataMediaType() == cloudevents.ApplicationXML

	style := a.defaultXSLT
	var xmlin *xml.XmlDocument
	var err error

	switch {
	case isStructuredTransform:
		req := &XSLTTransformStructuredRequest{}
		if err := event.DataAs(req); err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}

		style, err = parseXSLT(req.XSLT)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}

		xmlin, err = parseXML(req.XML)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}

	case isXML:
		xmlin, err = parseXML(string(event.DataEncoded))
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}

	default:
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("unexpected type or media-type for the incoming event"), nil)
	}

	output, err := style.Process(xmlin, xslt.StylesheetOptions{
		IndentOutput: true,
		Parameters:   nil})

	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			fmt.Errorf("eror processing XML with XSLT: %v", err), nil)
	}

	return a.replier.Ok(&event, []byte(output), targetce.ResponseWithDataContentType(cloudevents.ApplicationXML))
}

func parseXML(in string) (*xml.XmlDocument, error) {
	return xml.Parse([]byte(in), xml.DefaultEncodingBytes, nil, xml.StrictParseOption, xml.DefaultEncodingBytes)
}

func parseXSLT(in string) (*xslt.Stylesheet, error) {
	doc, err := parseXML(in)
	if err != nil {
		return nil, err
	}

	return xslt.ParseStylesheet(doc, "")
}
