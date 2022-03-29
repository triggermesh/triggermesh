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

package main

import (
	"fmt"
	"strings"
)

// diff stores information about differences between two YAML nodes.
type diff struct {
	objectName   string
	yamlNodeDesc string
	yamlDocLine  int
	diff         string
}

// String implements the fmt.Stringer interface.
func (e diff) String() string {
	return fmt.Sprintf("[%s] in node %q at line %d: %s",
		e.objectName, e.yamlNodeDesc, e.yamlDocLine, e.diff)
}

// diffsText returns a list of diffs as a single string.
func diffsText(dl []diff) string {
	if len(dl) == 0 {
		return ""
	}

	var output strings.Builder

	for i := 0; i < len(dl); i++ {
		if i > 0 {
			output.WriteByte('\n')
		}
		output.WriteString(dl[i].String())
	}

	return output.String()
}
