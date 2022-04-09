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

// AuthResponseData describes a set of data expected at the response of an Auth token request.
type AuthResponseData struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// ProcessResponseData describes a set of data expected at the response of a "Process" or "Release" key.
type ProcessResponseData struct {
	Count int `json:"@odata.count"`
	Value []struct {
		Key string `json:"Key"`
	} `json:"value"`
}

// RobotResponseData represents expected response data from robot calls.
type RobotResponseData struct {
	Count int `json:"@odata.count"`
	Value []struct {
		ID int `json:"Id"`
	} `json:"value"`
}

// StartJobData represents the expected payload to be provided to the adapter.
type StartJobData struct {
	InputArguments string `json:"InputArguments"`
}

// startInfo describes a set of data required to start a job
type startInfo struct {
	ReleaseKey     string `json:"ReleaseKey"`
	Strategy       string `json:"Strategy"`
	RobotIds       []int  `json:"RobotIds"`
	JobsCount      int    `json:"JobsCount"`
	Source         string `json:"Source"`
	InputArguments string `json:"InputArguments"`
}

// JobInfo describes a set of data required to start a job.
type JobInfo struct {
	startInfo `json:"startInfo"`
}

// QueuePostData represents queue data to be posted.
type QueuePostData struct {
	QueueItemData `json:"itemData"`
}

// QueueItemData represents the nested data expected in a QueuePostData struct.
type QueueItemData struct {
	Name            string            `json:"Name"`
	Priority        string            `json:"Priority"`
	SpecificContent map[string]string `json:"SpecificContent"`
}
