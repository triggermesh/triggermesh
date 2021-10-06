/*
Copyright 2020 TriggerMesh Inc.

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
	"runtime"

	"knative.dev/eventing/pkg/adapter/v2"

	"github.com/triggermesh/triggermesh/pkg/sources/adapter/awssqssource"
)

func main() {
	setMaxProcs(runtime.NumCPU())

	adapter.Main("awssqssource", awssqssource.NewEnvConfig, awssqssource.NewAdapter)
}

// setMaxProcs sets the number of threads that can be used by the current
// process.
//
// Knative uses uber-go/automaxprocs to automatically determine the number of
// threads (GOMAXPROCS) available to the process based on CPU quotas (e.g.
// 'cpu.request <= 1' translates to 1 thread, regardless of the number of
// physical cores).
// This event source has a very predictable CPU profile: it spends most of its
// time sending network requests without performing any computation on the
// results, so we assume the defined CPU limit allows all CPU cores to be used
// without starving on CPU time.
func setMaxProcs(procs int) int {
	return runtime.GOMAXPROCS(procs)
}
