// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

	_, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

	if err != nil {
		t.Fatalf("known paths: %v", err)
	}
	if asked.Method != http.MethodGet || asked.URL.Path != treePath ||
		asked.URL.RawQuery != "recursive=1" || asked.Header.Get("Authorization") != "Bearer "+forgeToken {
		t.Fatalf("asked %s %s?%s with %q, want GET %s?recursive=1 as the installation",
			asked.Method, asked.URL.Path, asked.URL.RawQuery, asked.Header.Get("Authorization"), treePath)
	}
}

var _ domain.RepositoryPaths = github.RepositoryPaths{}

const (
	treeRepository = "acme/payments"
	treePath       = "/repos/" + treeRepository + "/git/trees/HEAD"
)

func TestKnownPathsOfHoldsTheTreesFilesAndNoDirectoryOrSubmodule(t *testing.T) {
	server := serving(t, treePath, `{"truncated": false, "tree": [
		{"path": "internal", "type": "tree"},
		{"path": "internal/auth.go", "type": "blob"},
		{"path": "vendored", "type": "commit"}
	]}`)

	known, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

	if err != nil {
		t.Fatalf("known paths: %v", err)
	}
	for path, isHeld := range map[string]bool{"internal/auth.go": true, "internal": false, "vendored": false} {
		if known.Contains(repositoryPathOf(t, path)) != isHeld {
			t.Errorf("known paths hold %s: %t, want %t", path, !isHeld, isHeld)
		}
	}
}

func TestKnownPathsOfRefusesATreeHoldingAFileThatIsNotARepositoryPath(t *testing.T) {
	server := serving(t, treePath, `{"truncated": false, "tree": [
		{"path": "docs/release notes.md", "type": "blob"},
		{"path": "internal/auth.go", "type": "blob"}
	]}`)

	_, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("error is %v, want %v", err, domain.ErrPathOutsideRepository)
	}
}

func TestKnownPathsOfNeverParsesTheNameOfADirectoryOrSubmodule(t *testing.T) {
	server := serving(t, treePath, `{"truncated": false, "tree": [
		{"path": "docs/release notes", "type": "tree"},
		{"path": "internal/auth.go", "type": "blob"}
	]}`)

	_, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

	if err != nil {
		t.Fatalf("known paths: %v", err)
	}
}

func TestKnownPathsOfRefusesATreeTheForgeTruncated(t *testing.T) {
	server := serving(t, treePath,
		`{"truncated": true, "tree": [{"path": "internal/auth.go", "type": "blob"}]}`)

	_, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

	if !errors.Is(err, github.ErrTruncatedTree) {
		t.Fatalf("error is %v, want %v", err, github.ErrTruncatedTree)
	}
}

func TestKnownPathsOfReportsARepositoryTheForgeCannotFindAsNoSuchRepository(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)

	_, err := repositoryPathsAt(t, server).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

	if !errors.Is(err, github.ErrNoSuchRepository) {
		t.Fatalf("error is %v, want %v", err, github.ErrNoSuchRepository)
	}
}

func TestKnownPathsOfReadsATreeAtTheResponseBoundAndRefusesOneByteOver(t *testing.T) {
	const bound = 256
	listing := `{"truncated": false, "tree": [{"path": "internal/auth.go", "type": "blob"}]}`
	for padding, isRefused := range map[int]bool{bound - len(listing): false, bound - len(listing) + 1: true} {
		server := serving(t, treePath, listing+strings.Repeat(" ", padding))
		settings := settingsAt(server)
		settings.MaxResponseBytes = bound

		_, err := repositoryPathsWith(t, settings).KnownPathsOf(t.Context(), governedRepositoryOf(t, treeRepository))

		if errors.Is(err, github.ErrResponseTooLarge) != isRefused {
			t.Errorf("a tree of %d bytes against a bound of %d: error %v", len(listing)+padding, bound, err)
		}
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
	return repositoryPathsWith(t, settingsAt(server))
}

func repositoryPathsWith(t *testing.T, settings github.Settings) github.RepositoryPaths {
	t.Helper()
	paths, err := github.ParseRepositoryPaths(settings)
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
