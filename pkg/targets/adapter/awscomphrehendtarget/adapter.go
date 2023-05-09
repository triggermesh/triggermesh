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

package awscomphrehendtarget

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"knative.dev/pkg/logging"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehend"
	"github.com/aws/aws-sdk-go/service/comprehend/comprehendiface"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// NewTarget constructs a target's adapter.
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.AWSComprehendTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(env.Region).
		WithMaxRetries(5),
	))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAWSComprehendResult),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &comprehendAdapter{
		comprehend: comprehend.New(sess, config),

		language: env.Language,
		ceClient: ceClient,
		replier:  replier,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*comprehendAdapter)(nil)

type comprehendAdapter struct {
	language   string
	comprehend comprehendiface.ComprehendAPI

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Start implements pkgadapter.Adapter.
func (a *comprehendAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting The AWS Comprehend Target Adapter")
	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}

	return nil
}

func (a *comprehendAdapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	r := &Response{}
	var eventJSONMap map[string]interface{}
	var mixed, neg, pos float64
	var dSI comprehend.DetectSentimentInput
	dSI.SetLanguageCode(a.language)
	if err := event.DataAs(&eventJSONMap); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	for _, v := range eventJSONMap {
		str := fmt.Sprintf("%v", v)
		dSI.SetText(str)
		req, resp := a.comprehend.DetectSentimentRequest(&dSI)
		err := req.Send()
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}

		mixed = mixed + float64(*resp.SentimentScore.Mixed)
		neg = neg + float64(*resp.SentimentScore.Negative)
		pos = pos + float64(*resp.SentimentScore.Positive)
	}

	r.Positive = pos
	r.Mixed = mixed
	r.Negative = neg
	if pos > neg && pos > mixed {
		r.Result = "Positive"
	}

	if neg > pos && neg > mixed {
		r.Result = "Negative"
	}

	if (mixed > pos) && (mixed > neg) {
		r.Result = "mixed"
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err := responseEvent.SetData(cloudevents.ApplicationJSON, r)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	responseEvent.SetType(v1alpha1.EventTypeAWSComprehendResult)
	responseEvent.SetSource(event.ID())
	return &responseEvent, cloudevents.ResultACK
}
