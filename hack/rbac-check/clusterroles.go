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
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dprotaso/go-yit"
	"gopkg.in/yaml.v3"
	"knative.dev/pkg/codegen/cmd/injection-gen/generators"
)

const (
	clusterRolesFileName = "200-clusterroles.yaml"

	rbacCheckTag = "rbac-check"
)

// readClusterRoles returns the YAML documents contained in the given file.
// An error is returned if any decoded document doesn't describe a ClusterRole.
func readClusterRoles(file string) ([]*yaml.Node, error) {
	f, err := filesystem.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", file, err)
	}

	decoder := yaml.NewDecoder(f)

	var docs []*yaml.Node

	// Read all YAML documents contained in the file.
	for i := 0; ; i++ {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding YAML document: %w", err)
		}

		// perform sanity checks, even though Decode() shouldn't read
		// anything but YAML documents
		if doc.Kind != yaml.DocumentNode {
			return nil, fmt.Errorf("decoded YAML node %d in file %s has kind %s, expected only documents",
				i, file, doc.Tag)
		}
		if nNodes := len(doc.Content); nNodes != 1 {
			return nil, fmt.Errorf("decoded YAML node %d in file %s contains %d nodes instead of 1",
				i, file, nNodes)
		}

		// ensure the document actually contains a ClusterRole
		if _, ok := yit.FromNode(doc.Content[0]).Filter(clusterRole)(); !ok {
			return nil, fmt.Errorf("decoded YAML node %d in file %s doesn't represent a ClusterRole object",
				i, file)
		}

		docs = append(docs, &doc)
	}

	return docs, nil
}

// clusterRole is a yit.Predicate which matches Kubernetes objects of kind ClusterRole.
var clusterRole = yit.Intersect(
	// Predicate: node represents a Kubernetes API object in the RBAC API group
	yit.WithMapKeyValue(
		yit.WithStringValue("apiVersion"),
		yit.WithStringValue("rbac.authorization.k8s.io/v1"),
	),
	// Predicate: node represents a ClusterRole
	yit.WithMapKeyValue(
		yit.WithStringValue("kind"),
		yit.WithStringValue("ClusterRole"),
	),
)

// k8sObjectName extracts the name from given YAML document, if that node
// contains a Kubernetes API object.
func k8sObjectName(doc *yaml.Node) (string, error) {
	// iterate over content
	it := yit.FromNode(doc.Content[0]).
		// Access nested 'metadata' attribute
		ValuesForMap(
			yit.WithStringValue("metadata"),
			yit.WithKind(yaml.MappingNode),
		).
		// Access nested 'name' attribute
		ValuesForMap(
			yit.WithStringValue("name"),
			yit.StringValue,
		)

	// no need to iterate in a loop here, the StringValue predicate
	// guarantees we get a single node
	node, ok := it()
	if !ok || node.Value == "" {
		return "", errors.New("missing metadata.name attribute")
	}

	return node.Value, nil
}

// rulesNodesWithRBACCheckTag returns an iterator which visits all 'rules'
// sub-nodes tagged with '+rbac-check'.
// The given YAML document is expected to contain a ClusterRole object.
func rulesNodesWithRBACCheckTag(doc *yaml.Node) yit.Iterator {
	// iterate over content
	return yit.FromNode(doc.Content[0]).
		// Access nested 'rules' attribute
		ValuesForMap(
			yit.WithStringValue("rules"),
			yit.WithKind(yaml.SequenceNode),
		).
		Values().
		Filter(
			withRBACCheckTag,
		)
}

// apiGroupsNodeItems returns an iterator which visits all items of the
// 'apiGroups' sub-node of a ClusterRole's 'rules' node.
func apiGroupsNodeItems(node *yaml.Node) yit.Iterator {
	return yit.FromNode(node).
		// Access nested 'apiGroups' attribute
		ValuesForMap(
			yit.WithStringValue("apiGroups"),
			yit.WithKind(yaml.SequenceNode),
		).
		Values()
}

// resourcesNode returns an iterator which visits the 'resources' sub-node of a
// ClusterRole's 'rules' node.
func resourcesNode(node *yaml.Node) yit.Iterator {
	return yit.FromNode(node).
		// Access nested 'resources' attribute
		ValuesForMap(
			yit.WithStringValue("resources"),
			yit.WithKind(yaml.SequenceNode),
		)
}

// withRBACCheckTag is a yit.Predicate which matches on nodes tagged with '+rbac-check'.
func withRBACCheckTag(node *yaml.Node) bool {
	for _, line := range headCommentLines(node) {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "+"+rbacCheckTag) {
			return true
		}
	}
	return false
}

// extractRBACCheckTags parses '+rbac-check' tags from a YAML node's comments,
// and returns them in the format:
//
//   map[string][]string{
//     "subresource": {"status"},
//   }
//
// The returned map is empty if the parsed tag doesn't contain any key/value,
// nil if the provided YAML node isn't tagged.
func extractRBACCheckTags(node *yaml.Node) generators.CommentTag {
	commentLines := headCommentLines(node)

	tags := generators.ExtractCommentTags("+", commentLines)

	kv, isTagged := tags[rbacCheckTag]
	if !isTagged {
		return nil
	}
	if len(kv) == 0 {
		return make(generators.CommentTag)
	}

	return kv
}

// headCommentLines returns a YAML node's comment as a slice of lines.
func headCommentLines(node *yaml.Node) []string {
	if node.HeadComment == "" {
		return nil
	}

	var commentLines []string

	r := bufio.NewReader(strings.NewReader(node.HeadComment))
	for {
		line, err := r.ReadString('\n')

		commentLines = append(commentLines, strings.TrimPrefix(line, "#"))

		if err != nil {
			if err == io.EOF {
				break
			}
			// erroring while reading a string buffer is unlikely,
			// so we panic instead of returning the error to
			// simplify error handling in callers
			panic(fmt.Errorf("reading line from Reader: %w", err))
		}
	}

	return commentLines
}
