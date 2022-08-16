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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"

	"github.com/triggermesh/triggermesh/pkg/targets/adapter/salesforcetarget/auth"
	"go.uber.org/zap"
)

const (
	// base URL for Salesforce Lighting Platform
	salesforceAPIData = "/services/data/"
)

// salesforceVersion is returned from Salesforce when
// successfuly querying the data services endpoint.
type salesforceVersion struct {
	Version string
	Label   string
	URL     string
}

// SalesforceClient is the implementation of the Salesforce client
type SalesforceClient struct {
	auth  *auth.JWTAuthenticator
	creds *auth.Credentials

	apiVersion       string
	servicesDataPath string
	client           *http.Client
	logger           *zap.SugaredLogger
	mutex            sync.RWMutex
}

// Options for the Salesforce client
type Options func(*SalesforceClient)

// New creates a default Salesforce API client.
func New(authenticator *auth.JWTAuthenticator, logger *zap.SugaredLogger, opts ...Options) *SalesforceClient {
	sfc := &SalesforceClient{
		auth:   authenticator,
		logger: logger,
	}

	for _, opt := range opts {
		opt(sfc)
	}

	if sfc.client == nil {
		sfc.client = http.DefaultClient
	}

	return sfc
}

// WithAPIVersion sets a specific API version at the Salesforce client. If
// version is an empty string the client will choose latest upon authentication.
func WithAPIVersion(version string) Options {
	return func(c *SalesforceClient) {
		c.apiVersion = version
	}
}

// WithHTTPClient sets the HTTP client to be used.
func WithHTTPClient(httpClient *http.Client) Options {
	return func(c *SalesforceClient) {
		c.client = httpClient
	}
}

// Authenticate and performs checks regarding
// the Salesforce version
func (c *SalesforceClient) Authenticate(ctx context.Context) error {
	creds, err := c.auth.CreateOrRenewCredentials()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err != nil {
		// If there was an error with refresh, let's try to
		// use a new access token next time
		c.creds = nil
		return fmt.Errorf("could not perform authentication: %w", err)
	}
	c.creds = creds

	// If version is hardcoded
	if c.apiVersion != "" {
		c.servicesDataPath = path.Join(salesforceAPIData, c.apiVersion)
		return nil
	}

	// we already have the lock
	res, err := c.doCall(ctx, http.MethodGet, salesforceAPIData, nil, nil)
	if err != nil {
		return fmt.Errorf("could not retrieve available versions: %w", err)
	}

	if res.StatusCode != 200 {
		return c.manageAPIError(res)
	}

	var versions []salesforceVersion
	err = json.NewDecoder(res.Body).Decode(&versions)
	if err != nil {
		return fmt.Errorf("cannot decode Salesforce versions: %w", err)
	}

	var maxV float64 = 0
	for _, v := range versions {
		i, err := strconv.ParseFloat(v.Version, 64)
		if err != nil {
			// ignore errors and parse next version
			continue
		}
		if i > maxV {
			maxV = i
			c.servicesDataPath = v.URL
		}
	}

	return nil
}

// Do method will use the Salesforce API using the passed parameters
// adding only authentication header and the host. It is the caller responsability
// to add all toher elements to the call.
//
// This can be useful when a previous API call returned an URL that contains the
// full path to an element or a pagination.
func (c *SalesforceClient) Do(ctx context.Context, sfr SalesforceAPIRequest) (*http.Response, error) {
	return c.doRetriableCall(ctx, string(sfr.Action), path.Join(c.servicesDataPath, sfr.ResourcePath, sfr.ObjectPath, sfr.RecordPath), sfr.Query, []byte(sfr.Payload))
}

// doRetriableCall executes a Salesforce API call in a thread safe manner. If authentication fails
// the first time, it will authenticate and retry a second time.
func (c *SalesforceClient) doRetriableCall(ctx context.Context, method, urlPath string, query map[string]string, payload []byte) (res *http.Response, err error) {
	for i := 0; i < 2; i++ {
		res, err = c.doLockingCall(ctx, method, urlPath, query, payload)
		if err != nil || i != 0 || (res.StatusCode != http.StatusUnauthorized && res.StatusCode != http.StatusForbidden) {
			break
		}
		if err = c.Authenticate(ctx); err != nil {
			return
		}
	}
	return
}

// doLockingCall executes a Salesforce API call in a thread safe manner.
func (c *SalesforceClient) doLockingCall(ctx context.Context, method, urlPath string, query map[string]string, payload []byte) (*http.Response, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.doCall(ctx, method, urlPath, query, payload)
}

// doCall is not thread safe and should only be used if the client lock has been previously acquired.
func (c *SalesforceClient) doCall(ctx context.Context, method, urlPath string, query map[string]string, payload []byte) (*http.Response, error) {
	u, err := url.Parse(c.creds.InstanceURL)
	if err != nil {
		return nil, fmt.Errorf("base URL %q is not parseable: %w", c.creds.InstanceURL, err)
	}

	u.Path = path.Join(u.Path, urlPath)
	if len(query) > 0 {
		q := url.Values{}
		for k, v := range query {
			q.Add(k, v)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	if c.creds != nil {
		req.Header.Add("Authorization", "Bearer "+c.creds.Token)
	}

	req = req.WithContext(ctx)
	return c.client.Do(req)
}

type salesforceError struct {
	Fields    []string
	Message   string
	ErrorCode string
}

// doNonLockingCall is not thread safe and should only be used if the client lock has been previously acquired.
func (c *SalesforceClient) manageAPIError(res *http.Response) error {
	msg := fmt.Sprintf("API returned an error (%d): ", res.StatusCode)
	body := io.NopCloser(res.Body)

	// try to use the docummented Salesforce format.
	// See: https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/errorcodes.htm
	se := &salesforceError{}
	err := json.NewDecoder(body).Decode(se)
	if err == nil {
		return fmt.Errorf(msg+"(%s) %s. Fields %v", se.ErrorCode, se.Message, se.Fields)
	}

	// write raw response as a string
	b, err := io.ReadAll(res.Body)
	if err == nil {
		return fmt.Errorf(msg+"%s", string(b))
	}

	// last choice when all previous fail
	c.logger.Warnf("Could not read error message from API: %v", err)
	return fmt.Errorf(msg+"%s", string(b))
}
