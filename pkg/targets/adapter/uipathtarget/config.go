/*
Copyright (c) 2021 TriggerMesh Inc.

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

import pkgadapter "knative.dev/eventing/pkg/adapter/v2"

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	RobotName          string `envconfig:"UIPATH_ROBOT_NAME" required:"true"`
	ProcessName        string `envconfig:"UIPATH_PROCESS_NAME" required:"true"`
	TenantName         string `envconfig:"UIPATH_TENANT_NAME" required:"true"`
	AccountLogicalName string `envconfig:"UIPATH_ACCOUNT_LOGICAL_NAME" required:"true"`
	ClientID           string `envconfig:"UIPATH_CLIENT_ID" required:"true"`
	UserKey            string `envconfig:"UIPATH_USER_KEY" required:"true"`
	OrganizationUnitID string `envconfig:"UIPATH_ORGANIZATION_UNIT_ID" required:"true"`
}
