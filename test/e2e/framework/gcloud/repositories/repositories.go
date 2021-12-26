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
	"google.golang.org/api/sourcerepo/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateRepository creates a repository named after the given framework.Framework.
func CreateRepository(repoCli *sourcerepo.Service, project string, f *framework.Framework) *sourcerepo.Repo {
	repoRequest := &sourcerepo.Repo{
		Name: project + "/repos/" + f.UniqueName,
	}

	createRepo := repoCli.Projects.Repos.Create(project, repoRequest)

	repo, err := createRepo.Do()
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create repo %q: %s", f.UniqueName, err)
	}

	return repo
}

// DeleteRepository deletes a repository.
func DeleteRepository(repoCli *sourcerepo.Service, name string) {
	deleteRepo := repoCli.Projects.Repos.Delete(name)

	if _, err := deleteRepo.Do(); err != nil {
		framework.FailfWithOffset(2, "Failed to delete repo %q: %s", name, err)
	}
}
