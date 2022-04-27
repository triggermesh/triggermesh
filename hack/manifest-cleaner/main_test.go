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
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const testInput = `
apiVersion: test/v0
kind: Test
# +rbac-check:subresource=fake
metadata:
  name: test-1

---
---
# Copyright 2022 ACME Corp.
#
# Licensed under the Test License.
#


apiVersion: test/v0
kind: Test
metadata:
  # A nested comment
  #+rbac-check
  name: test-2
---
# Copyright 2022 ACME Corp.
#
# Licensed under the Test License.
#

---
apiVersion: test/v0
kind: Test
metadata:
  #     +rbac-check:subresource=fake
  name: test-3
`

const expectOutput = `
# Copyright 2022 ACME Corp.
#
# Licensed under the Test License.
#

apiVersion: test/v0
kind: Test
metadata:
  name: test-1
---
apiVersion: test/v0
kind: Test
metadata:
  # A nested comment
  name: test-2
---
apiVersion: test/v0
kind: Test
metadata:
  name: test-3
`

const testTimeout = 1 * time.Second

func TestRun(t *testing.T) {
	stdin := io.NopCloser(strings.NewReader(testInput))
	var stdout strings.Builder

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := run(ctx, stdin, &stdout); err != nil {
		t.Fatal("Unexpected runtime error:", err)
	}

	expectOutput := strings.TrimLeft(expectOutput, "\n")
	output := stdout.String()

	if d := cmp.Diff(expectOutput, output); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}
