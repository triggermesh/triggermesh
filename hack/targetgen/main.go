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

func main() {
	kind := flag.String("kind", "TestTarget", "specify the Target kind")
	cfgDir := flag.String("c", "../../",
		"Path of the directory containing the TriggerMesh deployment manifests")
	flag.Parse()
	*cfgDir = path.Clean(*cfgDir)
	temp := &component{}
	temp.Kind = strings.ToLower(*kind)
	temp.FullCaps = strings.ToUpper(*kind)

	// make cmd directory
	path := "cmd/" + temp.Kind + "-adapter"
	cmdPath := filepath.Join(*cfgDir, path)
	err := os.MkdirAll(cmdPath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// // make adapter directory
	path = "pkg/targets/adapter/" + temp.Kind
	adapterPath := filepath.Join(*cfgDir, path)
	err = os.MkdirAll(adapterPath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// make reconciler directory
	path = "pkg/targets/reconciler/" + temp.Kind
	reconcilerPath := filepath.Join(*cfgDir, path)
	err = os.MkdirAll(reconcilerPath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// populate cmd directory
	// read main.go and replace the template variables
	path = *cfgDir + "/cmd/" + temp.Kind + "-adapter/main.go"
	temp.replaceTemplates("scaffolding/cmd/newtarget-adapter/main.go", path)

	// populate adapter directory
	// read adapter.go
	path = *cfgDir + "/pkg/targets/adapter/" + temp.Kind + "/adapter.go"
	temp.replaceTemplates("scaffolding/pkg/targets/adapter/newtarget/adapter.go", path)

	// read newtarget_lifecycle.go
	path = *cfgDir + "/pkg/apis/targets/v1alpha1/" + temp.Kind + "_lifecycle.go"
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_lifecycle.go", path)

	// read newtarget_types.go
	path = *cfgDir + "/pkg/apis/targets/v1alpha1/" + temp.Kind + "_types.go"
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_types.go", path)

	// populate reconciler directory
	// read adapter.go
	path = *cfgDir + "/pkg/targets/reconciler/" + temp.Kind + "/adapter.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/adapter.go", path)

	// read controller_test.go
	path = *cfgDir + "/pkg/targets/reconciler/" + temp.Kind + "/controller_test.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller_test.go", path)

	// read controller.go
	path = *cfgDir + "/pkg/targets/reconciler/" + temp.Kind + "/controller.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller.go", path)

	// read reconciler.go
	path = *cfgDir + "/pkg/targets/reconciler/" + temp.Kind + "/reconciler.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/reconciler.go", path)

	// populate the config directory
	// read 301-newtarget.yaml.go
	path = *cfgDir + "/config/301-" + temp.Kind + ".yaml"
	temp.replaceTemplates("scaffolding/config/301-newtarget.yaml", path)

	fmt.Println("done")
	fmt.Println("Next Steps:")
	fmt.Println("Update `cmd/triggermesh-controller/main.go`")
	fmt.Println("Update `config/500-controller.yaml`")
	fmt.Println("Update `pkg/api/targets/v1alpha1/register.go`")
	fmt.Printf("Create kodata symlinks in cmd/%s", temp.Kind)
	fmt.Println("")
	fmt.Println("Run `make codegen`")
}

type component struct {
	Kind      string
	TitleCase string
	FullCaps  string
}

func (a *component) replaceTemplates(filename, outputname string) {
	tmp, err := template.ParseFiles(filename)
	if err != nil {
		fmt.Println(err)
	}

	file, err := os.Create(outputname)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file.Close()

	err = tmp.Execute(file, a)

	if err != nil {
		fmt.Println(err)
	}

}
