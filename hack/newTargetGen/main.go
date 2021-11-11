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

	// make cmd folder
	path := fmt.Sprintf("cmd/%s", temp.Name+"-adapter")
	newpath := filepath.Join("../../", path)
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// // make adapter folder
	path = fmt.Sprintf("pkg/targets/adapter/%s", temp.Name)
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// make reconciler folder
	path = fmt.Sprintf("pkg/targets/reconciler/%s", temp.Name)
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// populate cmd folder
	// read dockerfile and replace the template variables
	path = fmt.Sprintf("cmd/%s/Dockerfile", temp.Name+"-adapter")
	newpath = filepath.Join("../../", path)
	temp.replaceTemplates("scaffolding/cmd/newtarget-adapter/Dockerfile", newpath)

	// read main.go and replace the template variables
	path = fmt.Sprintf("../../cmd/%s/main.go", temp.Name+"-adapter")
	temp.replaceTemplates("scaffolding/cmd/newtarget-adapter/main.go", path)

	// populate adapter folder
	// read adapter.go
	path = fmt.Sprintf("../../pkg/targets/adapter/%s/adapter.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/targets/adapter/newtarget/adapter.go", path)

	// // read newtarget_lifecycle.go
	path = fmt.Sprintf("../../pkg/apis/targets/v1alpha1/%s_lifecycle.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_lifecycle.go", path)

	// // read newtarget_types.go
	path = fmt.Sprintf("../../pkg/apis/targets/v1alpha1/%s_types.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_types.go", path)

	// populate reconciler folder
	// read adapter.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/adapter.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/adapter.go", path)

	// read controller_test.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/controller_test.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller_test.go", path)

	// read controller.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/controller.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller.go", path)

	// read reconciler.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/reconciler.go", temp.Name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/reconciler.go", path)

	// populate the config folder
	// read 301-newtarget.yaml.go
	path = fmt.Sprintf("../../config/301-%s.yaml", temp.Name)
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
