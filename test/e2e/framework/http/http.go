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

// Package http contains helpers related to HTTP protocol
package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudevents/sdk-go/v2/event/datacodec/json"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// PostJSONRequest send an arbitrary JSON payload to an endpoint.
func PostJSONRequest(url string, payload interface{}) {
	p, err := json.Encode(context.Background(), payload)
	if err != nil {
		framework.FailfWithOffset(2, "Error encoding payload to JSON: %s", err)
	}

	res, err := http.Post(url, "application/json", bytes.NewBuffer(p))
	if err != nil {
		framework.FailfWithOffset(2, "Error POSTing to %s: %s", url, err)
	}

	if res.StatusCode >= 400 {
		framework.FailfWithOffset(2, "POSTing to %s returned error code %d", url, res.StatusCode)
	}
}

// PostJSONRequestWithRetries send an arbitrary JSON payload to an endpoint.
func PostJSONRequestWithRetries(interval, timeout time.Duration, url string, payload interface{}) {
	if err := wait.Poll(interval, timeout, postJSONRequestSucceed(url, payload)); err != nil {
		framework.FailfWithOffset(2, "Error POSTing to %s: %s", url, err)
	}
}

func postJSONRequestSucceed(url string, payload interface{}) wait.ConditionFunc {
	p, err := json.Encode(context.Background(), payload)
	if err != nil {
		framework.FailfWithOffset(2, "Error encoding payload to JSON: %s", err)
	}
	return func() (bool, error) {
		res, err := http.Post(url, "application/json", bytes.NewBuffer(p))
		if err != nil {
			return false, nil
		}
		if res.StatusCode >= 400 {
			return false, fmt.Errorf("POSTing to %s returned error code %d", url, res.StatusCode)
		}
		return true, nil
	}
}
