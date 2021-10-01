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

package hasuratarget

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

const baseURL = "/v1/graphql"

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	var token string
	queries := make(map[string]string)
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)
	isAdmin := false

	if env.JwtToken != "" {
		token = env.JwtToken
	} else if env.AdminToken != "" {
		token = env.AdminToken
		isAdmin = true
	}

	if env.Queries != "" {
		err := json.Unmarshal([]byte(env.Queries), &queries)
		if err != nil {
			logger.Panicw("Unable to extract queries", zap.Error(err))
		}
	}

	return &hasuraAdapter{
		endpoint:    env.Endpoint,
		token:       token,
		isAdmin:     isAdmin,
		defaultRole: env.DefaultRole,
		queries:     queries,

		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*hasuraAdapter)(nil)

type hasuraAdapter struct {
	endpoint    string
	token       string
	isAdmin     bool
	defaultRole string
	queries     map[string]string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	client   *http.Client
}

type hasuraTargetPayload struct {
	Query         string            `json:"query"`
	OperationName string            `json:"operationName,omitempty"`
	Variables     map[string]string `json:"variables,omitempty"`
}

// Start setup the http client that will communicate with the Hasura endpoint
func (h *hasuraAdapter) Start(ctx context.Context) error {
	h.logger.Info("Starting Hasura adapter")

	if h.token != "" && !h.isAdmin {
		src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: h.token})
		h.client = oauth2.NewClient(ctx, src)
	} else {
		h.client = http.DefaultClient
	}

	return h.ceClient.StartReceiver(ctx, h.dispatch)
}

// dispatch accepts the CloudEvent as a query, and sends the results back as a
// response CloudEvent.
func (h *hasuraAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var data []byte

	// If the Ce-Type matches org.graphql.query, then Ce-Subject must be
	// set to reflect the query to pass, and the data must contain a map of
	// key/value pairs representing variables. Note: the Ce-Subject _must_
	// match one of the queries defined in the target spec.
	//
	// If the Ce-Type matches org.graphql.query.raw, the data is expected
	// to contain a valid GraphQL query which conforms to the
	// hasuraTargetPayload schema.
	//
	switch typ := event.Type(); typ {

	case v1alpha1.EventTypeHasuraQuery:
		keys := make(map[string]string)
		if err := event.DataAs(&keys); err != nil {
			return h.reportError("Unable to extract keys from payload", err)
		}

		query := h.queries[event.Subject()]
		if query == "" {
			return h.reportError("Unknown query: "+event.Subject(), nil)
		}

		payload := &hasuraTargetPayload{
			Query:         query,
			OperationName: event.Subject(),
			Variables:     keys,
		}

		jsonEvent, err := json.Marshal(payload)
		if err != nil {
			return h.reportError("Unable to create new payload", err)
		}

		data = jsonEvent

	case v1alpha1.EventTypeHasuraQueryRaw:
		data = event.Data()

	default:
		return h.reportError("Unsupported event type "+strconv.Quote(typ), nil)
	}

	req, err := http.NewRequest("POST", h.endpoint+baseURL, bytes.NewReader(data))
	if err != nil {
		return h.reportError("Unable to build request", err)
	}

	if h.token != "" && h.isAdmin {
		req.Header.Add("x-hasura-admin-secret", h.token)
	}

	if h.defaultRole != "" {
		req.Header.Add("x-hasura-role", h.defaultRole)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return h.reportError("Unable to send request", err)
	}

	// NOTE: All responses will return a 200 OK message as per https://hasura.io/docs/1.0/graphql/core/api-reference/graphql-api/index.html

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return h.reportError("Unable to read response body", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, respBody)
	if err != nil {
		return h.reportError("error generating response event", err)
	}

	responseEvent.SetType(v1alpha1.EventTypeHasuraResult)
	responseEvent.SetSource(h.endpoint)

	return &responseEvent, cloudevents.ResultACK
}

func (h *hasuraAdapter) reportError(msg string, err error) (*cloudevents.Event, cloudevents.Result) {
	h.logger.Errorw(msg, zap.Error(err))

	return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, msg)
}
