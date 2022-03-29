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

package common

import (
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

const envAWSEndpointURL = "AWS_ENDPOINT_URL"

// EndpointResolver returns a custom endpoints.Resolver which allows users to
// target API-compatible alternatives to the public AWS cloud.
func EndpointResolver(partition string, opts ...func(*endpoints.Options)) endpoints.Resolver {
	rslvr := func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if endpointURLStr := os.Getenv(envAWSEndpointURL); endpointURLStr != "" {
			endpointURL, err := url.Parse(endpointURLStr)
			if err != nil {
				return endpoints.ResolvedEndpoint{}, fmt.Errorf("invalid AWS endpoint URL: %w", err)
			}

			return endpoints.ResolvedEndpoint{
				URL:         endpointURL.String(),
				PartitionID: partition,
			}, nil
		}

		return endpoints.DefaultResolver().EndpointFor(service, region, opts...)
	}

	return endpoints.ResolverFunc(rslvr)
}
