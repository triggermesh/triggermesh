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

package opentelemetrytarget

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/metric/sdkapi"
	"go.opentelemetry.io/otel/metric/unit"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type Instrument struct {
	sdkapi.Descriptor
	Name        string
	Instrument  string
	Number      string
	Description string
}

type Instruments []*Instrument

// MAINTENANCE: instrumentKinds needs to be kept in sync with
// sdkapi.InstrumentKind enum.
var instrumentKinds = map[string]sdkapi.InstrumentKind{
	"Histogram":     sdkapi.HistogramInstrumentKind,
	"Counter":       sdkapi.CounterInstrumentKind,
	"UpDownCounter": sdkapi.UpDownCounterInstrumentKind,

	// Comment async instruments, they are not supported
	// "GaugeObserver":         sdkapi.GaugeObserverInstrumentKind,
	// "CounterObserver":       sdkapi.CounterObserverInstrumentKind,
	// "UpDownCounterObserver": sdkapi.UpDownCounterObserverInstrumentKind,
}

// MAINTENANCE: numberKinds needs to be kept in sync with
// number.Kind enum.
var numberKinds = map[string]number.Kind{
	"Int64":   number.Int64Kind,
	"Float64": number.Float64Kind,
}

func (is *Instruments) Decode(value string) error {
	if err := json.Unmarshal([]byte(value), is); err != nil {
		return err
	}

	for _, i := range *is {
		if i.Name == "" {
			return errors.New("metrics must include a name")
		}
		if i.Instrument == "" {
			return errors.New("metrics must include an instrument kind")
		}
		if i.Number == "" {
			return errors.New("metrics must include a number kind")
		}

		ikind, ok := instrumentKinds[i.Instrument]
		if !ok {
			return fmt.Errorf("unknown metric instrument kind %q", i.Instrument)
		}

		nkind, ok := numberKinds[i.Number]
		if !ok {
			return fmt.Errorf("unknown metric number kind %q", i.Number)
		}

		i.Descriptor = sdkapi.NewDescriptor(i.Name, ikind, nkind, i.Description, unit.Dimensionless)
	}

	return nil
}

type envAccessor struct {
	pkgadapter.EnvConfig

	// Cortex connection parameters.
	CortexEndpoint      string        `envconfig:"OPENTELEMETRY_CORTEX_ENDPOINT"`
	CortexRemoteTimeout time.Duration `envconfig:"OPENTELEMETRY_CORTEX_REMOTE_TIMEOUT" default:"30s"`
	CortexBearerToken   string        `envconfig:"OPENTELEMETRY_CORTEX_BEARER_TOKEN"`
	CortexPushInterval  time.Duration `envconfig:"OPENTELEMETRY_CORTEX_PUSH_INTERVAL" default:"10s"`

	// OpenTelemetry instruments information.
	Instruments Instruments `envconfig:"OPENTELEMETRY_INSTRUMENTS" required:"true"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
}
