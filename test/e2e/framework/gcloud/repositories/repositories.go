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

// Package repositories contains helpers for Google Cloud Repositories.
package repositories

import (
	"errors"
	"net/http"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/sourcerepo/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateRepository creates a repository named after the given framework.Framework.
func CreateRepository(repoCli *sourcerepo.Service, project string, f *framework.Framework) *sourcerepo.Repo {
	repoRequest := &sourcerepo.Repo{
		Name: "projects/" + project + "/repos/" + f.UniqueName,
	}

	createRepo := repoCli.Projects.Repos.Create("projects/"+project, repoRequest)

	repo, err := createRepo.Do()
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create repo %q: %s", f.UniqueName, err)
	}

	return repo
}

// DeleteRepository deletes a repository by resource name.
// It doesn't fail if the repository doesn't exist.
func DeleteRepository(repoCli *sourcerepo.Service, name string) {
	deleteRepository(repoCli, name, true)
}

// MustDeleteRepository deletes a repository by resource name.
// Unlike DeleteRepository, the delete call fails if the repository isn't found.
func MustDeleteRepository(repoCli *sourcerepo.Service, name string) {
	deleteRepository(repoCli, name, false)
}

func deleteRepository(repoCli *sourcerepo.Service, name string, tolerateNotFound bool) {
	_, err := repoCli.Projects.Repos.Delete(name).Do()
	switch {
	case isNotFound(err) && tolerateNotFound:
		return
	case err != nil:
		framework.FailfWithOffset(3, "Failed to delete repo %q: %s", name, err)
	}
}

func isNotFound(err error) bool {
	if gapiErr := (*googleapi.Error)(nil); errors.As(err, &gapiErr) {
		return gapiErr.Code == http.StatusNotFound
	}

	return false
}
