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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// readInput reads from the given Reader until EOF and returns the read data.
func readInput(ctx context.Context, in io.ReadCloser) ([]byte, error) {
	var data []byte
	var err error

	doneCh := make(chan (struct{}))
	defer close(doneCh)

	go func() {
		data, err = io.ReadAll(in)
		doneCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		// In some implementations, Close() may preempt a Read().
		// This doesn't seem to be the case with Files (e.g. STDIN),
		// but we close the input nevertheless for good measure.
		if err := in.Close(); err != nil {
			return nil, fmt.Errorf("closing input: %w", err)
		}
		return nil, nil

	case <-doneCh:
		return data, err
	}
}

// writeNodes marshals the given nodes to a sequence of YAML documents
// prepended with the given header and writes the result to out.
func writeNodes(out io.Writer, nodes []*yaml.Node, header string) error {
	var b bytes.Buffer

	if header := strings.TrimSpace(header); header != "" {
		b.WriteString(header)
		b.WriteByte('\n')
		b.WriteByte('\n')
	}

	e := yaml.NewEncoder(&b)
	e.SetIndent(2)

	for i, n := range nodes {
		if err := e.Encode(n); err != nil {
			return fmt.Errorf("writing YAML encoding of node %d: %w", i, err)
		}
	}

	if _, err := out.Write(b.Bytes()); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}
