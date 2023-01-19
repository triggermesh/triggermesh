/*
Copyright 2023 TriggerMesh Inc.

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

package mongodbtarget

import pkgadapter "knative.dev/eventing/pkg/adapter/v2"

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig
	// ServerURL is the connection string to the MongoDB server.
	// ex: mongodb+srv://<user>:<password>@<database_url>/myFirstDatabase.
	ServerURL string `envconfig:"MONGODB_SERVER_URL" required:"true"`
	// DefaultDatabase defines a default database to interact with. If another
	// database is specified in an incoming event, the event-specified database will
	// be used in the request, effectively overwriting this one.
	DefaultDatabase string `envconfig:"MONGODB_DEFAULT_DATABASE" required:"true"`
	// DefaultCollection defines a default collection to interact with. If another
	// collection is specified in an incoming event, the event-specified collection will
	// be used in the request, effectively overwriting this one.
	DefaultCollection string `envconfig:"MONGODB_DEFAULT_COLLECTION" required:"true"`
	// BridgeIdentifier is the name of the bridge workflow this target is part of.
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization.
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
}
