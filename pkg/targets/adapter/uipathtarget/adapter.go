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

package uipathtarget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget returns the adapter implementation.
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.UiPathTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	return &uipathAdapter{
		client:             http.DefaultClient,
		robotName:          env.RobotName,
		processName:        env.ProcessName,
		tenantName:         env.TenantName,
		accountLogicalName: env.AccountLogicalName,
		clientID:           env.ClientID,
		userKey:            env.UserKey,
		organizationUnitID: env.OrganizationUnitID,
		ceClient:           ceClient,
		logger:             logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*uipathAdapter)(nil)

type uipathAdapter struct {
	client             *http.Client
	organizationUnitID string
	robotName          string
	processName        string
	tenantName         string
	accountLogicalName string
	clientID           string
	userKey            string
	ceClient           cloudevents.Client
	logger             *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *uipathAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting UiPath adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *uipathAdapter) dispatch(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	a.logger.Debug("Processing event")

	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeUiPathQueuePost:
		if err := a.initiateQueueStart(event); err != nil {
			return fmt.Errorf("error posting to queue: %w", err)
		}

	case v1alpha1.EventTypeUiPathStartJob:
		if err := a.initiateJobStart(event); err != nil {
			return fmt.Errorf("error starting a job: %w", err)
		}

	default:
		return fmt.Errorf("event type %q is not supported at the UiPath target", typ)
	}

	return cloudevents.ResultACK
}

func (a *uipathAdapter) initiateQueueStart(event cloudevents.Event) error {
	qd := &QueueItemData{}
	if err := event.DataAs(qd); err != nil {
		return err
	}

	bearer, err := a.getAccessToken()
	if err != nil {
		return err
	}

	return a.postToQueue(qd, bearer)
}

func (a *uipathAdapter) initiateJobStart(event cloudevents.Event) error {
	jd := &StartJobData{}
	if err := event.DataAs(jd); err != nil {
		return err
	}

	bearer, err := a.getAccessToken()
	if err != nil {
		return err
	}

	robotID, err := a.getRobotID(bearer)
	if err != nil {
		return err
	}

	releaseKey, err := a.getReleaseKey(bearer)
	if err != nil {
		return err
	}

	return a.startJob(bearer, releaseKey, jd.InputArguments, robotID)
}

func (a *uipathAdapter) getAccessToken() (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     a.clientID,
		"refresh_token": a.userKey,
	})
	if err != nil {
		return "", fmt.Errorf("marshaling request for retrieving an access token: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, "https://account.uipath.com/oauth/token", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("creating request for auth tokwn: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-UIPATH-TenantName", a.tenantName)
	res, err := a.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("processing auth request: %w", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body from requesting the auth token: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got HTTP code %d while retrieving access token", res.StatusCode)
	}

	rd := &AuthResponseData{}
	if err := json.Unmarshal(body, rd); err != nil {
		return "", err
	}

	return rd.AccessToken, nil
}

func (a *uipathAdapter) getReleaseKey(bearer string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, a.makeReleaseURL(), nil)
	if err != nil {
		return "", fmt.Errorf("creating request to retrieve release key: %w", err)
	}

	request.Header.Set("X-UIPATH-OrganizationUnitId", a.organizationUnitID)
	request.Header.Set("Authorization", "Bearer "+bearer)
	res, err := a.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("processing release key request: %w", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("retrieving release key: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got HTTP code %d while retrieving release key", res.StatusCode)
	}

	var rd = &ProcessResponseData{}
	if err := json.Unmarshal(body, rd); err != nil {
		return "", err
	}

	return rd.Value[0].Key, nil
}

func (a *uipathAdapter) getRobotID(bearer string) (int, error) {
	request, err := http.NewRequest(http.MethodGet, a.makeRobotIDURL(), nil)
	if err != nil {
		return -1, fmt.Errorf("creating request to retrieve robot ID: %w", err)
	}

	request.Header.Set("X-UIPATH-OrganizationUnitId", a.organizationUnitID)
	request.Header.Set("Authorization", "Bearer "+bearer)
	res, err := a.client.Do(request)
	if err != nil {
		return -1, fmt.Errorf("processing robot ID request: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, fmt.Errorf("retrieving robot ID: %w", err)
	}

	defer res.Body.Close()

	var rd = &RobotResponseData{}
	if err := json.Unmarshal(body, rd); err != nil {
		return -1, fmt.Errorf("unmarshaling JSON response: %w", err)
	}

	if rd.Count > 0 {
		return rd.Value[0].ID, nil
	}

	return 0, nil
}

func (a *uipathAdapter) postToQueue(qd *QueueItemData, bearer string) error {
	qpd := &QueuePostData{
		QueueItemData: *qd,
	}

	b, err := json.Marshal(qpd)
	if err != nil {
		return fmt.Errorf("marshaling request for posting to a queue: %w", err)
	}

	a.logger.Debug("Sending request to post to a queue: ", string(b))

	req, err := http.NewRequest(http.MethodPost, a.makeQueueURL(), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-UIPATH-OrganizationUnitId", a.organizationUnitID)
	req.Header.Set("Authorization", "Bearer "+bearer)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if a.logger.Desugar().Core().Enabled(zap.DebugLevel) {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			a.logger.Debugw("Failed to read response body from posting to a queue", zap.Error(err))
		} else {
			a.logger.Debug("Request to post to a queue returned: ", string(body))
		}
	}

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("got HTTP code %d while posting to queue", res.StatusCode)
	}

	return nil
}

func (a *uipathAdapter) startJob(bearer, rK, inputArgs string, robotID int) error {
	r := &JobInfo{
		startInfo: startInfo{
			ReleaseKey:     rK,
			Strategy:       "Specific",
			RobotIds:       []int{robotID},
			JobsCount:      0,
			Source:         "Manual",
			InputArguments: inputArgs,
		},
	}

	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshaling request for starting a job: %w", err)
	}

	a.logger.Debug("Sending request to start a job: ", string(b))

	req, err := http.NewRequest(http.MethodPost, a.makeStartJobURL(), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("creating job request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-UIPATH-OrganizationUnitId", a.organizationUnitID)
	req.Header.Set("Authorization", "Bearer "+bearer)

	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("starting job: %w", err)
	}

	defer res.Body.Close()

	if a.logger.Desugar().Core().Enabled(zap.DebugLevel) {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			a.logger.Debugw("Failed to read response body from starting a job", zap.Error(err))
		} else {
			a.logger.Debug("Request to start a job returned: ", string(body))
		}
	}

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("got HTTP code %d while starting job", res.StatusCode)
	}

	return nil
}

func (a *uipathAdapter) makeStartJobURL() string {
	return "https://cloud.uipath.com/" + a.accountLogicalName + "/" + a.tenantName + "/odata/Jobs/UiPath.Server.Configuration.OData.StartJobs"
}

func (a *uipathAdapter) makeQueueURL() string {
	return "https://cloud.uipath.com/" + a.accountLogicalName + "/" + a.tenantName + "/odata/Queues/UiPathODataSvc.AddQueueItem"
}

func (a *uipathAdapter) makeReleaseURL() string {
	return "https://cloud.uipath.com/" + a.accountLogicalName + "/" + a.tenantName + "/odata/Releases" + `?$filter=%20Name%20eq%20'` + a.processName + "'"
}

func (a *uipathAdapter) makeRobotIDURL() string {
	return "https://cloud.uipath.com/" + a.accountLogicalName + "/" + a.tenantName + "/odata/Robots?$filter=Name%20eq%20'" + a.robotName + "'"
}
