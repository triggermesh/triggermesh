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

func main() {
	var name string
	var uppercaseName string
	// var capsName string

	fmt.Print("Enter the LOWERCASE VERSION of the target name: ")
	fmt.Scanf("%s", &name)
	fmt.Print("Enter the UPPERCASE VERSION of the target name: ")
	fmt.Scanf("%s", &uppercaseName)
	// fmt.Print("Enter the ALL CAPS VERISON of the target name: ")
	// fmt.Scanf("%s", &capsName)
	// TODO add naming validation here
	fmt.Printf("Lets make a %s target!", uppercaseName)

	// make cmd folder
	path := fmt.Sprintf("cmd/%s", name)
	newpath := filepath.Join("../../", path)
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// // make adapter folder
	path = fmt.Sprintf("pkg/targets/adapter/%s", name)
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// make reconciler folder
	path = fmt.Sprintf("pkg/targets/reconciler/%s", name)
	newpath = filepath.Join("../../", path)
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	// populate cmd folder
	// read dockerfile and replace the template variables
	data, err := ioutil.ReadFile("scaffolding/cmd/newtarget-adapter/Dockerfile")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper := bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase := bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("cmd/%s/Dockerfile", name)
	newpath = filepath.Join("../../", path)
	file, err := os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read main.go and replace the template variables
	data, err = ioutil.ReadFile("scaffolding/cmd/newtarget-adapter/main.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("cmd/%s/main.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// populate adapter folder

	// read adapter.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/adapter/newtarget/adapter.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/adapter/%s/adapter.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read newtarget_lifecycle.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/adapter/newtarget/newtarget_lifecycle.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/adapter/%s/%s_lifecycle.go", name, name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read newtarget_types.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/adapter/newtarget/newtarget_types.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/adapter/%s/%s_types.go", name, name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// populate reconciler folder
	// read adapter.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/reconciler/newtarget/adapter.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/reconciler/%s/adapter.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read controller_test.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/reconciler/newtarget/controller_test.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/reconciler/%s/controller_test.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read controller.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/reconciler/newtarget/controller.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/reconciler/%s/controller.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read reconciler_test.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/reconciler/newtarget/reconciler_test.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/reconciler/%s/reconciler_test.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	// read reconciler.go
	data, err = ioutil.ReadFile("scaffolding/pkg/targets/reconciler/newtarget/reconciler.go")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	filterUpper = bytes.ReplaceAll(data, []byte("$TARGETFULLCASE"), []byte(uppercaseName))
	filterLowercase = bytes.ReplaceAll(filterUpper, []byte("$TARGET"), []byte(name))

	path = fmt.Sprintf("pkg/targets/reconciler/%s/reconciler.go", name)
	newpath = filepath.Join("../../", path)
	file, err = os.Create(newpath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()
	_, err = file.Write(filterLowercase)
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	fmt.Println("done")

}
