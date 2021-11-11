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
	"path/filepath"
	"strings"
	"text/template"
)

type component struct {
	Name          string
	UppercaseName string
	FullCaps      string
}

func (a *component) replaceTemplates(filename, outputname string) {
	tmp1, err := template.ParseFiles(filename)
	if err != nil {
		fmt.Println(err)
	}

	file, err := os.Create(outputname)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file.Close()

	err = tmp1.Execute(file, a)

	if err != nil {
		fmt.Println(err)
	}

}

func main() {
	kind := flag.String("kind", "TestTarget", "specify the Target kind")
	flag.Parse()
	temp := &component{}
	temp.Name = strings.ToLower(*kind)
	temp.FullCaps = strings.ToUpper(*kind)

	// make cmd directory
	path := "cmd/" + temp.Name + "-adapter"
	newpath := filepath.Join("../../", path)
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// // make adapter directory
	path = "pkg/targets/adapter/%s" + temp.Name
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// make reconciler directory
	path = "pkg/targets/reconciler/" + temp.Name
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// populate cmd directory
	// read main.go and replace the template variables
	path = ".../../cmd/" + temp.Name + "-adapter/main.go"
	temp.replaceTemplates("scaffolding/cmd/newtarget-adapter/main.go", path)

	// populate adapter directory
	// read adapter.go
	path = ".../../pkg/targets/adapter/" + temp.Name + "/adapter.go"
	temp.replaceTemplates("scaffolding/pkg/targets/adapter/newtarget/adapter.go", path)

	// read newtarget_lifecycle.go
	path = "../../pkg/apis/targets/v1alpha1/" + temp.Name + "_lifecycle.go"
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_lifecycle.go", path)

	// read newtarget_types.go
	path = "../../pkg/apis/targets/v1alpha1/" + temp.Name + "_types.go"
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_types.go", path)

	// populate reconciler directory
	// read adapter.go
	path = "../../pkg/targets/reconciler/" + temp.Name + "/adapter.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/adapter.go", path)

	// read controller_test.go
	path = "../../pkg/targets/reconciler/" + temp.Name + "/controller_test.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller_test.go", path)

	// read controller.go
	path = "../../pkg/targets/reconciler/" + temp.Name + "/controller.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller.go", path)

	// read reconciler.go
	path = "../../pkg/targets/reconciler/" + temp.Name + "/reconciler.go"
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/reconciler.go", path)

	// populate the config directory
	// read 301-newtarget.yaml.go
	path = "../../config/301-" + temp.Name + ".yaml"
	temp.replaceTemplates("scaffolding/config/301-newtarget.yaml", path)

	fmt.Println("done")
	fmt.Println("Next Steps:")
	fmt.Println("Update `cmd/triggermesh-controller/main.go`")
	fmt.Println("Update `config/500-controller.yaml`")
	fmt.Println("Update `pkg/api/targets/v1alpha1/register.go`")
	fmt.Printf("Create kodata symlinks in cmd/%s", temp.Name)
	fmt.Println("")
	fmt.Println("Run `make codegen`")
}
