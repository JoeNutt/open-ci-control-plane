// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const publishedSchema = "schema/ci.diff.v1.schema.json"

func TestEveryEmittedDocumentMatchesThePublishedSchema(t *testing.T) {
	documents := map[string]domain.Document{
		"a change with nothing to report": {},
		"a change over two directories": domain.ParseDocument(
			domain.PullRequest{
				Number: 412, Repository: "acme/payments", BaseRef: "main",
				HeadSHA: "9f1cabc", Draft: true,
				Author: domain.Author{
					Login: "jo", Association: domain.AuthorAssociationMember},
				Labels:    domain.Declared[string]{"backend"},
				Approvals: domain.Declared[domain.Approval]{{Login: "sam"}},
			},
			domain.Change{Files: boundedFor(t,
				domain.File{Path: "internal/auth/middleware.go",
					Action:    domain.FileActionModified,
					Additions: 12, Deletions: 3, PatchBytes: 1840},
				domain.File{Path: "infra/iam.tf", Action: domain.FileActionAdded,
					Additions: 35, PatchBytes: 7280},
			)}, domain.ClassificationRules{}),
		"a change whose Go file declares imports at head": domain.ParseDocument(
			domain.PullRequest{Number: 412, Repository: "acme/payments", HeadSHA: "9f1cabc"},
			changeImporting(t, domain.File{
				Path: "internal/auth/middleware.go", Action: domain.FileActionModified},
				"context", "github.com/acme/payments/internal/session"),
			domain.ClassificationRules{}),
		"a change that adds and removes an edge": domain.ParseDocument(
			domain.PullRequest{Number: 412, Repository: "acme/payments", HeadSHA: "9f1cabc"},
			changeMovingAnEdge(t, domain.File{
				Path: "internal/change/domain/gate.go", Action: domain.FileActionModified}),
			domain.ClassificationRules{}),
	}

	for form, document := range documents {
		t.Run(form, func(t *testing.T) {
			for _, fault := range schemaFaultsIn(t, document) {
				t.Errorf("%s", fault)
			}
		})
	}
}

func changeImporting(t *testing.T, file domain.File, imports ...string) domain.Change {
	t.Helper()

	return domain.Change{
		Files:     boundedFor(t, file),
		Structure: structureOf(t, domain.ImportedFile{File: file, Head: fileImportsOf(t, imports...)}),
	}
}

func changeMovingAnEdge(t *testing.T, file domain.File) domain.Change {
	t.Helper()

	return domain.Change{
		Files: boundedFor(t, file),
		Structure: structureOf(t, domain.ImportedFile{
			File: file,
			Base: fileImportsOf(t, "context", "github.com/acme/payments/internal/forge"),
			Head: fileImportsOf(t, "context", "github.com/acme/payments/internal/session"),
		}),
	}
}

func boundedFor(t *testing.T, files ...domain.File) domain.BoundedFiles {
	t.Helper()

	bounded, err := domain.ParseBoundedFiles(files, domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("bound %d files: %v", len(files), err)
	}
	return bounded
}

func schemaFaultsIn(t *testing.T, document domain.Document) []string {
	t.Helper()

	matched, faults := matchAgainstPublishedSchema(t, askedOf(t, document))
	if matched {
		return nil
	}
	return faults
}

func askedOf(t *testing.T, document domain.Document) []byte {
	t.Helper()

	published, err := os.ReadFile(filepath.Join(moduleRoot(t), publishedSchema))
	if err != nil {
		t.Fatalf("read %s: %v", publishedSchema, err)
	}
	asked, err := json.Marshal(map[string]any{
		"document": document,
		"schema":   json.RawMessage(published),
	})
	if err != nil {
		t.Fatalf("marshal document and schema: %v", err)
	}
	return asked
}

func matchAgainstPublishedSchema(t *testing.T, asked []byte) (bool, []string) {
	t.Helper()

	matching := exec.Command("opa", "eval", "-I", "--format=raw",
		`json.match_schema(input.document, input.schema)`)
	matching.Stdin = bytes.NewReader(asked)
	answered, err := matching.CombinedOutput()
	if err != nil {
		t.Fatalf("opa eval against %s: %v\n%s", publishedSchema, err, answered)
	}

	var pair []json.RawMessage
	if err := json.Unmarshal(answered, &pair); err != nil || len(pair) != 2 {
		t.Fatalf("opa answered %s", answered)
	}
	var matched bool
	if err := json.Unmarshal(pair[0], &matched); err != nil {
		t.Fatalf("decode the match: %v", err)
	}
	var reported []struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(pair[1], &reported); err != nil {
		t.Fatalf("decode the faults: %v", err)
	}
	faults := make([]string, 0, len(reported))
	for _, fault := range reported {
		faults = append(faults, fault.Error)
	}
	return matched, faults
}
