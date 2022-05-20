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
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

/* A command which verifies that all custom types included in the TriggerMesh
   platform are referenced by the relevant RBAC roles.
*/

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	exitCode, err := run(ctx, os.Args, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running command: %s\n", err)
	}

	os.Exit(exitCode)
}

// Exit codes returned by the command.
const (
	exitCodeSuccess = 0
	exitCodeError   = 1
	exitCodeDiffs   = 3 // exit code 2 is already reserved by the 'flags' package
)

// filesystem is the fs.ReadDirFs used by the command to read configuration files.
// It is set as a global variable so that tests can override it with an
// alternative implementation.
var filesystem fs.ReadDirFS = (*osFS)(nil)

// run executes the command with the given arguments.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) (int /*exit code*/, error) {
	cmdName := path.Base(args[0])

	flags := flag.NewFlagSet(cmdName, flag.ExitOnError)
	flags.SetOutput(stderr)

	opts, err := readOpts(flags, args)
	if err != nil {
		return exitCodeError, fmt.Errorf("reading options: %w", err)
	}

	diffs, err := computeDiffs(*opts.cfgDir)
	if err != nil {
		return exitCodeError, fmt.Errorf("computing differences: %w", err)
	}

	exitCode := exitCodeSuccess
	if len(diffs) > 0 {
		exitCode = exitCodeDiffs
		fmt.Fprintln(stdout, diffsText(diffs))
		fmt.Fprintf(stderr, "Found %d inconsistencies\n", len(diffs))
	}

	return exitCode, nil
}

// computeDiffs returns the differences between the existing and expected
// ClusterRoles in the given config directory.
func computeDiffs(cfgDir string) ([]diff, error) {
	components, err := readComponents(cfgDir)
	if err != nil {
		return nil, fmt.Errorf("reading manifests from config directory %q: %w", cfgDir, err)
	}

	clusterRolesFilePath := path.Join(cfgDir, clusterRolesFileName)
	docs, err := readClusterRoles(clusterRolesFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading manifests for ClusterRoles: %w", err)
	}

	var diffs []diff

	for i, doc := range docs {
		name, err := k8sObjectName(doc)
		if err != nil {
			return nil, fmt.Errorf("reading name of decoded ClusterRole %d: %w", i, err)
		}

		it := rulesNodesWithRBACCheckTag(doc)

		for ruleNode, ok := it(); ok; ruleNode, ok = it() {
			tags := extractRBACCheckTags(ruleNode)
			if tags == nil {
				// This shouldn't happen.
				// Fail loudly if such node passed the filter despite the withRBACCheckTag predicate.
				panic(fmt.Errorf("encountered a rules node without tag while iterating over a list " +
					"that was expected to only contain tagged rules nodes"))
			}

			var resources []string

			it := apiGroupsNodeItems(ruleNode)
			for apiGroupNode, ok := it(); ok; apiGroupNode, ok = it() {
				resources = append(resources, components[apiGroupNode.Value]...)
			}

			// no need to iterate in a loop here, the iterator
			// accesses a single attribute
			it = resourcesNode(ruleNode)
			resourcesNode, ok := it()
			if !ok {
				return nil, fmt.Errorf("encountered a rule without resources in ClusterRole %q: %w",
					name, err)
			}

			expectResourcesNode := &yaml.Node{
				Kind:    yaml.SequenceNode,
				Tag:     "!!seq",
				Content: make([]*yaml.Node, 0, len(resourcesNode.Content)),
			}

			for _, res := range resources {
				val := res + "s" // pluralize kind
				if subRes, ok := tags["subresource"]; ok {
					if nVals := len(subRes); nVals != 1 {
						return nil, fmt.Errorf("encountered a tag with %d value(s) for the "+
							"subresource key: %v", nVals, subRes)
					}
					val += "/" + subRes[0]
				}

				expectResourcesNode.Content = append(expectResourcesNode.Content, &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: val,
				})
			}

			currentYAML, err := yaml.Marshal(*resourcesNode)
			if err != nil {
				return nil, fmt.Errorf("marshaling resources node at line %d for ClusterRole %s: %w",
					resourcesNode.Line, name, err)
			}

			expectYAML, err := yaml.Marshal(*expectResourcesNode)
			if err != nil {
				return nil, fmt.Errorf("marshaling resources node at line %d for ClusterRole %s: %w",
					resourcesNode.Line, name, err)
			}

			if diffStr := cmp.Diff(string(currentYAML), string(expectYAML)); diffStr != "" {
				diffs = append(diffs, diff{
					objectName:   name,
					yamlNodeDesc: "resources",
					yamlDocLine:  resourcesNode.Line,
					diff:         "(-:expect, +:got)\n" + diffStr,
				})
			}
		}
	}

	return diffs, nil
}

// cmdOpts are the options that can be passed to the command.
type cmdOpts struct {
	// path of the directory containing the TriggerMesh deployment manifests
	cfgDir *string
}

// readOpts parses and validates options from commmand-line flags.
func readOpts(f *flag.FlagSet, args []string) (*cmdOpts, error) {
	opts := &cmdOpts{}

	opts.cfgDir = f.String("c", "./config/",
		"Path of the directory containing the TriggerMesh deployment manifests")

	if err := f.Parse(args[1:]); err != nil {
		return nil, err
	}

	*opts.cfgDir = path.Clean(*opts.cfgDir)

	return opts, nil
}
