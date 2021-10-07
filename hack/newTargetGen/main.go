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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type template struct {
	name          string
	uppercaseName string
}

func (a *template) replaceTemplates(filename, outputname string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper := bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(a.uppercaseName))
	filterLowercase := bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(a.name))

	file, err := os.Create(outputname)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}
}

func main() {
	temp := &template{}
	// var capsName string

	fmt.Print("Enter the LOWERCASE VERSION of the target name: ")
	fmt.Scanf("%s", &temp.name)
	fmt.Print("Enter the UPPERCASE VERSION of the target name: ")
	fmt.Scanf("%s", &temp.uppercaseName)
	// fmt.Print("Enter the ALL CAPS VERISON of the target name: ")
	// fmt.Scanf("%s", &capsName)
	// TODO add naming validation here
	fmt.Printf("Lets make a %s target!", temp.uppercaseName)

	// make cmd folder
	path := fmt.Sprintf("cmd/%s", temp.name)
	newpath := filepath.Join("../../", path)
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// // make adapter folder
	path = fmt.Sprintf("pkg/targets/adapter/%s", temp.name)
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// make reconciler folder
	path = fmt.Sprintf("pkg/targets/reconciler/%s", temp.name)
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// populate cmd folder
	// read dockerfile and replace the template variables
	path = fmt.Sprintf("cmd/%s/Dockerfile", temp.name)
	newpath = filepath.Join("../../", path)
	temp.replaceTemplates("scaffolding/cmd/newtarget-adapter/Dockerfile", newpath)

	// read main.go and replace the template variables
	path = fmt.Sprintf("../../cmd/%s/main.go", temp.name)
	temp.replaceTemplates("scaffolding/cmd/newtarget-adapter/main.go", path)

	// populate adapter folder
	// read adapter.go
	path = fmt.Sprintf("../../pkg/targets/adapter/%s/adapter.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/targets/adapter/newtarget/adapter.go", path)

	// // read newtarget_lifecycle.go
	path = fmt.Sprintf("../../pkg/apis/targets/v1alpha1/%s_lifecycle.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_lifecycle.go", path)

	// // read newtarget_types.go
	path = fmt.Sprintf("../../pkg/apis/targets/v1alpha1/%s_types.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/apis/targets/v1alpha1/newtarget_types.go", path)

	// populate reconciler folder
	// read adapter.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/adapter.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/adapter.go", path)

	// read controller_test.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/controller_test.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller_test.go", path)

	// read controller.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/controller.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/controller.go", path)

	// read reconciler_test.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/reconciler_test.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/reconciler_test.go", path)

	// read reconciler.go
	path = fmt.Sprintf("../../pkg/targets/reconciler/%s/reconciler.go", temp.name)
	temp.replaceTemplates("scaffolding/pkg/targets/reconciler/newtarget/reconciler.go", path)

	// populate the config folder
	// read 301-newtarget.yaml.go
	path = fmt.Sprintf("../../config/301-%s.yaml", temp.name)
	temp.replaceTemplates("scaffolding/config/301-newtarget.yaml", path)

	fmt.Println("")
	fmt.Println("done")

}
