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

package jiratarget

import (
	"context"
	"io/ioutil"
	"net/url"
	"path"

	"github.com/andygrunwald/go-jira"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// NewTarget creates a Jira target adapter
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)
	env := envAcc.(*envAccessor)

	basicAuth := jira.BasicAuthTransport{
		Username: env.JiraAuthUser,
		Password: env.JiraAuthToken,
	}

	jiraClient, err := jira.NewClient(basicAuth.Client(), env.JiraURL)
	if err != nil {
		logger.Panicw("Could not create the Jira client", zap.Error(err))
	}

	return &jiraAdapter{
		ceClient: ceClient,
		logger:   logger,

		jiraClient: jiraClient,
		baseURL:    env.JiraURL,
		resSource:  env.Namespace + "/" + env.Name + ": " + env.JiraURL,
	}
}

type jiraAdapter struct {
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	baseURL    string
	jiraClient *jira.Client
	resSource  string
}

var _ pkgadapter.Adapter = (*jiraAdapter)(nil)

func (a *jiraAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Jira adapter")

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}
	return nil
}

func (a *jiraAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {

	switch event.Type() {
	case v1alpha1.EventTypeJiraIssueCreate:
		return a.jiraIssueCreate(ctx, event)
	case v1alpha1.EventTypeJiraIssueGet:
		return a.jiraIssueGet(ctx, event)
	case v1alpha1.EventTypeJiraCustom:
		return a.jiraCustomRequest(ctx, event)
	}

	a.logger.Errorf("Event type %q is not supported", event.Type())
	return nil, cloudevents.ResultNACK
}

func (a *jiraAdapter) jiraIssueCreate(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	j := &jira.Issue{}
	if err := event.DataAs(j); err != nil {
		a.logger.Errorw("Error processing incoming event data as Jira Issue", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	issue, res, err := a.jiraClient.Issue.CreateWithContext(ctx, j)
	if err != nil {
		respErr := jira.NewJiraError(res, err)
		a.logger.Errorw("Error requesting Jira API", zap.Error(respErr))
		return nil, cloudevents.ResultACK
	}

	out := cloudevents.NewEvent()
	if err := out.SetData(cloudevents.ApplicationJSON, issue); err != nil {
		a.logger.Errorw("Error generating response event", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	out.SetID(uuid.New().String())
	out.SetType(v1alpha1.EventTypeJiraIssue)
	out.SetSource(a.resSource)

	return &out, cloudevents.ResultACK
}

func (a *jiraAdapter) jiraIssueGet(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	j := &IssueGetRequest{}
	if err := event.DataAs(j); err != nil {
		a.logger.Errorw("Error processing incoming event data as IssueGetRequest", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	issue, res, err := a.jiraClient.Issue.GetWithContext(ctx, j.ID, &j.Options)
	if err != nil {
		respErr := jira.NewJiraError(res, err)
		a.logger.Errorw("Error requesting Jira API", zap.Error(respErr))
		return nil, cloudevents.ResultACK
	}

	out := cloudevents.NewEvent()
	if err := out.SetData(cloudevents.ApplicationJSON, issue); err != nil {
		a.logger.Errorw("Error generating response event", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	out.SetID(uuid.New().String())
	out.SetType(v1alpha1.EventTypeJiraIssue)
	out.SetSource(a.resSource)

	return &out, cloudevents.ResultACK
}

func (a *jiraAdapter) jiraCustomRequest(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	j := &JiraAPIRequest{}
	if err := event.DataAs(j); err != nil {
		a.logger.Errorw("Error processing incoming event data as generic Jira API request", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	u, err := url.Parse(a.baseURL)
	if err != nil {
		a.logger.Errorw("Error parsing base URL", zap.Error(err))
		return nil, cloudevents.ResultACK
	}
	u.Path = path.Join(u.Path, j.Path)

	if len(j.Query) > 0 {
		q := url.Values{}
		for k, v := range j.Query {
			q.Add(k, v)
		}
		u.RawQuery = q.Encode()
	}

	req, err := a.jiraClient.NewRequestWithContext(ctx, string(j.Method), u.String(), j.Payload)
	if err != nil {
		a.logger.Errorw("Error creating request", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	res, err := a.jiraClient.Do(req, nil)
	if err != nil {
		respErr := jira.NewJiraError(res, err)
		a.logger.Errorw("Error requesting Jira API", zap.Error(respErr))
		return nil, cloudevents.ResultACK
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		a.logger.Errorw("Error reading response from Jira API", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	out := cloudevents.NewEvent()
	if err := out.SetData(cloudevents.ApplicationJSON, resBody); err != nil {
		a.logger.Errorw("Error generating response event", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	out.SetID(uuid.New().String())
	out.SetType(event.Type() + ".response")
	out.SetSource(a.resSource)

	return &out, cloudevents.ResultACK
}
