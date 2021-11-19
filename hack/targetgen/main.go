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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	apiVersion = "v1alpha1"

	cmdPath        = "cmd"
	configPath     = "config"
	templatesPath  = "scaffolding"
	apisPath       = "pkg/apis/targets"
	adapterPath    = "pkg/targets/adapter"
	reconcilerPath = "pkg/targets/reconciler"
)

type component struct {
	Kind          string
	LowercaseKind string
	UppercaseKind string
}

func main() {
	kind := flag.String("kind", "TestTarget", "Specify the Target kind")
	cfgDir := flag.String("config", "../../",
		"Path of the directory containing the TriggerMesh deployment manifests")
	flag.Parse()
	*cfgDir = path.Clean(*cfgDir)
	temp := &component{
		Kind:          *kind,
		LowercaseKind: strings.ToLower(*kind),
		UppercaseKind: strings.ToUpper(*kind),
	}

	// make cmd directory
	mustMkdirAll(filepath.Join(*cfgDir, cmdPath, temp.LowercaseKind+"-adapter"))

	// make adapter directory
	mustMkdirAll(filepath.Join(*cfgDir, adapterPath, temp.LowercaseKind))

	// make reconciler directory
	mustMkdirAll(filepath.Join(*cfgDir, reconcilerPath, temp.LowercaseKind))

	// populate cmd directory
	// read main.go and replace the template variables
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, cmdPath, "newtarget-adapter", "main.go"),
		filepath.Join(*cfgDir, cmdPath, temp.LowercaseKind+"-adapter/main.go"),
	); err != nil {
		log.Fatalf("failed creating the cmd templates: %v", err)
	}

	// populate adapter directory
	// read adapter.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, adapterPath, "newtarget", "adapter.go"),
		filepath.Join(*cfgDir, adapterPath, temp.LowercaseKind, "/adapter.go"),
	); err != nil {
		log.Fatalf("failed creating the adapter templates: %v", err)
	}

	// read newtarget_lifecycle.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, apisPath, apiVersion, "newtarget_lifecycle.go"),
		filepath.Join(*cfgDir, apisPath, apiVersion, temp.LowercaseKind+"_lifecycle.go"),
	); err != nil {
		log.Fatalf("failed creating the newtarget_lifecycle.go template: %v", err)
	}

	// read newtarget_types.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, apisPath, apiVersion, "newtarget_types.go"),
		filepath.Join(*cfgDir, apisPath, apiVersion, temp.LowercaseKind+"_types.go"),
	); err != nil {
		log.Fatalf("failed creating the newtarget_types.go template: %v", err)
	}

	// populate reconciler directory
	// read adapter.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, reconcilerPath, "newtarget", "adapter.go"),
		filepath.Join(*cfgDir, reconcilerPath, temp.LowercaseKind, "adapter.go"),
	); err != nil {
		log.Fatalf("failed creating the reconciler templates: %v", err)
	}

	// read controller_test.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, reconcilerPath, "newtarget", "controller_test.go"),
		filepath.Join(*cfgDir, reconcilerPath, temp.LowercaseKind, "controller_test.go"),
	); err != nil {
		log.Fatalf("failed creating the controller_test.go template: %v", err)
	}

	// read controller.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, reconcilerPath, "newtarget", "controller.go"),
		filepath.Join(*cfgDir, reconcilerPath, temp.LowercaseKind, "controller.go"),
	); err != nil {
		log.Fatalf("failed creating the controller.go template: %v", err)
	}

	// read reconciler.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, reconcilerPath, "newtarget", "reconciler.go"),
		filepath.Join(*cfgDir, reconcilerPath, temp.LowercaseKind, "reconciler.go"),
	); err != nil {
		log.Fatalf("failed creating the reconciler.go template: %v", err)
	}

	// populate the config directory
	// read 301-newtarget.yaml.go
	if err := temp.replaceTemplates(
		filepath.Join(templatesPath, configPath, "301-newtarget.yaml"),
		filepath.Join(*cfgDir, configPath, "301-"+temp.LowercaseKind+".yaml"),
	); err != nil {
		log.Fatalf("failed creating the CRD from the template: %v", err)
	}

	fmt.Println("done")
	fmt.Println("Next Steps:")
	fmt.Println("Update `cmd/triggermesh-controller/main.go`")
	fmt.Println("Update `config/500-controller.yaml`")
	fmt.Println("Update `pkg/apis/targets/v1alpha1/register.go`")
	fmt.Printf("Create kodata symlinks in cmd/%s", temp.LowercaseKind)
	fmt.Println("")
	fmt.Println("Run `make codegen`")
}

func (a *component) replaceTemplates(filename, outputname string) error {
	tmp, err := template.ParseFiles(filename)
	if err != nil {
		return err
	}

	file, err := os.Create(outputname)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmp.Execute(file, a)
}

func mustMkdirAll(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatalf("failed creating directory: %v", err)
	}
}
