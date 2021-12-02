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

package synchronizer

import (
	"fmt"
	"math/rand"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
)

// Correlation Key parameters.
const (
	defaultCorrelationKeyLen = 24
	maxCorrelationKeyLen     = 64

	correlationKeycharset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

var (
	// CloudEvent attributes cannot be used as a correltaion key.
	restrictedKeys = []string{
		"id",
		"type",
		"time",
		"subject",
		"schemaurl",
		"dataschema",
		"specversion",
		"datamediatype",
		"datacontenttype",
		"datacontentencoding",
	}

	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// CloudEventKey is the Correlation Key for the CloudEvents context.
type CloudEventKey struct {
	Attribute string
	Length    int
}

// NewCorrelationKey returns an instance of the CloudEvent Correlation key.
func NewCorrelationKey(attribute string, length int) (*CloudEventKey, error) {
	if length < 0 || length > maxCorrelationKeyLen {
		return nil, fmt.Errorf("correlation ID length must be in the range between 0 and %d, got %d", maxCorrelationKeyLen, length)
	}

	for _, rk := range restrictedKeys {
		if attribute == rk {
			return nil, fmt.Errorf("%q cannot be used as a correlation key", attribute)
		}
	}

	return &CloudEventKey{
		Attribute: attribute,
		Length:    length,
	}, nil
}

// Get returns the value of Correlation Key.
func (k *CloudEventKey) Get(event cloudevents.Event) (string, bool) {
	if val, exists := event.Extensions()[k.Attribute]; exists {
		return val.(string), true
	}
	return "", false
}

// Set updates the CloudEvent's context with the random Correlation Key value.
func (k *CloudEventKey) Set(event *cloudevents.Event) string {
	var correlationID string
	if k.Length == 0 {
		correlationID = randString(defaultCorrelationKeyLen)
	} else {
		correlationID = randString(k.Length)
	}
	event.SetExtension(k.Attribute, correlationID)
	return correlationID
}

// randString generates the random string with fixed length.
func randString(length int) string {
	k := make([]byte, length)
	l := len(correlationKeycharset)
	for i := range k {
		k[i] = correlationKeycharset[seededRand.Intn(l)]
	}
	return string(k)
}
