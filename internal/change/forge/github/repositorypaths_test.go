// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestKnownPathsOfAsksForTheDefaultBranchTreeAsTheInstallation(t *testing.T) {
	var asked *http.Request
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		asked = request
		write(t, writer, `{"truncated": false, "tree": []}`)
	})

	_, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, "acme/payments"))

	if err != nil {
		t.Fatalf("known paths: %v", err)
	}
	if asked.Method != http.MethodGet || asked.URL.Path != "/repos/acme/payments/git/trees/HEAD" ||
		asked.URL.RawQuery != "recursive=1" || asked.Header.Get("Authorization") != "Bearer "+forgeToken {
		t.Fatalf("asked %s %s?%s with %q, want GET /repos/acme/payments/git/trees/HEAD?recursive=1 as the installation",
			asked.Method, asked.URL.Path, asked.URL.RawQuery, asked.Header.Get("Authorization"))
	}
}

func TestKnownPathsOfHoldsTheTreesFilesAndNoDirectoryOrSubmodule(t *testing.T) {
	server := serving(t, "/repos/acme/payments/git/trees/HEAD", `{"truncated": false, "tree": [
		{"path": "internal", "type": "tree"},
		{"path": "internal/auth.go", "type": "blob"},
		{"path": "vendored", "type": "commit"}
	]}`)

	known, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, "acme/payments"))

	if err != nil {
		t.Fatalf("known paths: %v", err)
	}
	for path, isHeld := range map[string]bool{"internal/auth.go": true, "internal": false, "vendored": false} {
		if known.Contains(repositoryPathOf(t, path)) != isHeld {
			t.Errorf("known paths hold %s: %t, want %t", path, !isHeld, isHeld)
		}
	}
}

func TestKnownPathsOfHoldsEveryFileAnSFLineCouldNameAlongsideOneItCouldNot(t *testing.T) {
	server := serving(t, "/repos/acme/payments/git/trees/HEAD", `{"truncated": false, "tree": [
		{"path": "docs/release notes.md", "type": "blob"},
		{"path": "internal/auth.go", "type": "blob"}
	]}`)

	known, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, "acme/payments"))

	if err != nil {
		t.Fatalf("known paths: %v", err)
	}
	if !known.Contains(repositoryPathOf(t, "internal/auth.go")) {
		t.Error("a file beside one no SF line could name is not held")
	}
}

func repositoryPathOf(t *testing.T, stated string) domain.RepositoryPath {
	t.Helper()
	path, err := domain.ParseRepositoryPath(stated)
	if err != nil {
		t.Fatalf("parse repository path: %v", err)
	}
	return path
}

func repositoryPathsAt(t *testing.T, server *httptest.Server) github.RepositoryPaths {
	t.Helper()
	paths, err := github.ParseRepositoryPaths(settingsAt(server))
	if err != nil {
		t.Fatalf("parse repository paths: %v", err)
	}
	return paths
}

func governedRepositoryOf(t *testing.T, stated string) domain.GovernedRepository {
	t.Helper()
	repository, err := domain.ParseGovernedRepository(stated)
	if err != nil {
		t.Fatalf("parse governed repository: %v", err)
	}
	return repository
}
