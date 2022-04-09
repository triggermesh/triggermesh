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

package httptarget

import (
	"errors"
	"fmt"
	"strings"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	EventType   string `envconfig:"HTTP_EVENT_TYPE" required:"true"`
	EventSource string `envconfig:"HTTP_EVENT_SOURCE" required:"true"`

	URL               string            `envconfig:"HTTP_URL" required:"true"`
	Method            string            `envconfig:"HTTP_METHOD" required:"true"`
	SkipVerify        bool              `envconfig:"HTTP_SKIP_VERIFY"`
	CACertificate     string            `envconfig:"HTTP_CA_CERTIFICATE"`
	Headers           map[string]string `envconfig:"HTTP_HEADERS"`
	BasicAuthUsername string            `envconfig:"HTTP_BASICAUTH_USERNAME"`
	BasicAuthPassword string            `envconfig:"HTTP_BASICAUTH_PASSWORD"`

	OAuthClientID     string   `envconfig:"HTTP_OAUTH_CLIENT_ID"`
	OAuthClientSecret string   `envconfig:"HTTP_OAUTH_CLIENT_SECRET"`
	OAuthAuthTokenURL string   `envconfig:"HTTP_OAUTH_TOKEN_URL"`
	OAuthScopes       []string `envconfig:"HTTP_OAUTH_SCOPE"`
}

func (e *envAccessor) validateAuth() error {
	bAuth := e.isBasicAuth()
	oAuth := e.isOAuth()
	if bAuth && oAuth {
		return errors.New("cannot configure Basic Authentication and OAuth at the same time")
	}

	if !oAuth {
		return nil
	}

	requiredFields := map[string]string{
		"HTTP_OAUTH_CLIENT_ID":     e.OAuthClientID,
		"HTTP_OAUTH_CLIENT_SECRET": e.OAuthClientSecret,
		"HTTP_OAUTH_TOKEN_URL":     e.OAuthAuthTokenURL,
	}

	var missingFields []string

	for k, v := range requiredFields {
		if v == "" {
			missingFields = append(missingFields, k)
		}
	}

	if len(missingFields) != 0 {
		return fmt.Errorf("missing required OAuth fields %s", strings.Join(missingFields, ","))
	}

	return nil
}

func (e *envAccessor) isBasicAuth() bool {
	return e.BasicAuthUsername != "" || e.BasicAuthPassword != ""
}

func (e *envAccessor) isOAuth() bool {
	return e.OAuthClientID != "" || e.OAuthClientSecret != "" ||
		e.OAuthAuthTokenURL != "" || len(e.OAuthScopes) != 0
}
