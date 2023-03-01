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

// Package repositories contains helpers for Google Cloud Repositories.
package repositories

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/sourcerepo/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

const (
	gitDefaultRemoteName = "origin"
	gitDefaultBranch     = "master"

	gitCommitterName  = "TriggerMesh e2e"
	gitCommitterEmail = "dev@triggermesh.com"
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
func DeleteRepository(repoCli *sourcerepo.Service, repoName string) {
	_, err := repoCli.Projects.Repos.Delete(repoName).Do()
	if err != nil && !isNotFound(err) {
		framework.FailfWithOffset(2, "Failed to delete repo %q: %s", repoName, err)
	}
}

func isNotFound(err error) bool {
	if gapiErr := (*googleapi.Error)(nil); errors.As(err, &gapiErr) {
		return gapiErr.Code == http.StatusNotFound
	}

	return false
}

// InitRepoAndCommit initializes the source repository with the given URL, and
// pushes a commit to it.
// It assumes a running environment with a pre-authenticated Google Cloud SDK (gcloud CLI).
func InitRepoAndCommit(repoURL string) {
	tmpdir, err := os.MkdirTemp("", "tme2e-gcloudsrcrepo")
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tmpdir)

	runCmdInDir(tmpdir, "git", "init", ".")
	// Ensures 'git' commands authenticate with the remote using 'git git-credential-gcloud.sh'
	// ref. https://git-scm.com/docs/gitcredentials#_custom_helpers
	runCmdInDir(tmpdir, "git", "config", "credential."+urlSchemeAndHost(repoURL)+".helper", "gcloud.sh")

	runCmdInDir(tmpdir, "git", "remote", "add", gitDefaultRemoteName, repoURL)
	runCmdInDir(tmpdir, "git", "fetch", gitDefaultRemoteName)

	runCmdInDir(tmpdir, "git", "config", "user.name", gitCommitterName)
	runCmdInDir(tmpdir, "git", "config", "user.email", gitCommitterEmail)

	f, err := os.Create(filepath.Join(tmpdir, "README.md"))
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create file: %s", err)
	}
	func() {
		defer f.Close()

		fContent := []byte("File updated at " + time.Now().Format(time.RFC3339Nano))
		if _, err := f.Write(fContent); err != nil {
			framework.FailfWithOffset(2, "Failed to write to file: %s", err)
		}
	}()

	runCmdInDir(tmpdir, "git", "add", ".")
	runCmdInDir(tmpdir, "git", "commit", "-m", "Update README.md", "--no-gpg-sign")
	defaultBranch := strings.TrimSpace(runCmdInDirAndReturnOutput(tmpdir, "git", "config", "--global", "--get", "init.defaultBranch"))
	if defaultBranch == "" {
		defaultBranch = gitDefaultBranch
	}
	runCmdInDir(tmpdir, "git", "push", "--set-upstream", gitDefaultRemoteName, defaultBranch)

}

func runCmdInDir(dir, name string, args ...string) {
	_ = runCmdInDirAndReturnOutput(dir, name, args...)
}

func runCmdInDirAndReturnOutput(dir, name string, args ...string) string {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr, cmd.Stdout = &stderr, &stdout

	if err := cmd.Run(); err != nil {
		if stderr := stderr.String(); stderr != "" {
			err = fmt.Errorf(stderr+": %w", err)
		}

		framework.FailfWithOffset(3, "Failed to run command %q: %s",
			strings.Join(append([]string{name}, args...), " "), err)
		return ""
	}

	return stdout.String()

}

func urlSchemeAndHost(fullURL string) string {
	u, err := url.Parse(fullURL)
	if err != nil {
		framework.FailfWithOffset(3, "Failed to parse URL: %s", err)
	}

	u = &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   "/",
	}
	return u.String()
}
