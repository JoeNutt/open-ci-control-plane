// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const catalogPath = "catalog/clean-architecture.yaml"

func TestPolicyContentsReadsAPolicyFileAtThePolicyCommitAsTheInstallation(t *testing.T) {
	var asked recordedRequest
	const catalog = "schema_version: 1\n"
	server := recording(t, &asked, fileAnswer(catalog))
	contents, err := github.ParsePolicyContents(settingsAt(server))
	if err != nil {
		t.Fatalf("parse policy contents: %v", err)
	}
	path, err := domain.ParseRepositoryPath(catalogPath)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}

	read, err := contents.ContentsAt(context.Background(), policyHeadCommit(t), path)

	if err != nil {
		t.Fatalf("contents at: %v", err)
	}
	if string(read) != catalog {
		t.Errorf("read %q, want %q", read, catalog)
	}
	want := recordedRequest{
		method:        http.MethodGet,
		escapedPath:   policyRepositoryPath + "/contents/" + catalogPath,
		rawQuery:      "ref=" + policyHead,
		authorization: "Bearer " + forgeToken,
	}
	if asked != want {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}
