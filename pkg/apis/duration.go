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

package apis

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration extends time.Duration with additional methods for (de-)serialization to/from
// JSON, allowing it to be embedded in custom API objects.
type Duration time.Duration

var (
	_ fmt.Stringer     = (*Duration)(nil)
	_ json.Marshaler   = (*Duration)(nil)
	_ json.Unmarshaler = (*Duration)(nil)
)

// String implements the fmt.Stringer interface.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	dur, err := time.ParseDuration(dataStr)
	if err != nil {
		return fmt.Errorf("failed to parse duration %q: %w", dataStr, err)
	}

	*d = Duration(dur)

	return nil
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}
