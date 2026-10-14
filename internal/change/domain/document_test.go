// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestZeroDocumentEmitsEmptyPathsAsList(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}
	if !strings.Contains(string(encoded), `"paths":[]`) {
		t.Fatalf(`want "paths":[] in %s`, encoded)
	}
}

func TestDocumentDeclaresSchemaVersion(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}

	var decoded struct {
		SchemaVersion *int `json:"schema_version"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	if decoded.SchemaVersion == nil {
		t.Fatalf("schema_version absent from %s", encoded)
	}
	if *decoded.SchemaVersion != 1 {
		t.Fatalf("schema_version is %d, want 1", *decoded.SchemaVersion)
	}
}

func TestPullRequestContextIsSiblingOfDiff(t *testing.T) {
	document := domain.Document{PR: domain.PullRequest{
		Number:     412,
		Repository: "acme/payments",
		BaseRef:    "main",
		HeadSHA:    "9f1c",
	}}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}

	var decoded struct {
		Diff *json.RawMessage `json:"diff"`
		PR   *struct {
			Number     int    `json:"number"`
			Repository string `json:"repository"`
			BaseRef    string `json:"base_ref"`
			HeadSHA    string `json:"head_sha"`
		} `json:"pr"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	if decoded.Diff == nil {
		t.Fatalf("diff absent from %s", encoded)
	}
	if decoded.PR == nil {
		t.Fatalf("pr absent from %s, so it is not a sibling of diff", encoded)
	}
	if decoded.PR.Number != 412 || decoded.PR.Repository != "acme/payments" ||
		decoded.PR.BaseRef != "main" || decoded.PR.HeadSHA != "9f1c" {
		t.Fatalf("pr carries %+v, want the supplied context", *decoded.PR)
	}
}

func TestZeroDocumentPopulatesEveryPullRequestField(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}

	var decoded struct {
		PR struct {
			Draft     *bool              `json:"draft"`
			Labels    *[]string          `json:"labels"`
			Approvals *[]json.RawMessage `json:"approvals"`
			Author    *struct {
				Login       *string `json:"login"`
				Association *string `json:"association"`
				IsBot       *bool   `json:"is_bot"`
			} `json:"author"`
		} `json:"pr"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}

	pr := decoded.PR
	if pr.Draft == nil || pr.Labels == nil || pr.Approvals == nil || pr.Author == nil {
		t.Fatalf("pr omits a declared field in %s", encoded)
	}
	if len(*pr.Labels) != 0 || len(*pr.Approvals) != 0 {
		t.Fatalf("pr lists are not empty in %s", encoded)
	}
	if pr.Author.Login == nil || pr.Author.Association == nil || pr.Author.IsBot == nil {
		t.Fatalf("pr.author omits a declared field in %s", encoded)
	}
}

func TestZeroDocumentPopulatesEveryDiffField(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}

	var decoded struct {
		Diff struct {
			Paths  *[]string          `json:"paths"`
			Files  *[]json.RawMessage `json:"files"`
			Kinds  *json.RawMessage   `json:"kinds"`
			Totals *struct {
				Files      *int `json:"files"`
				Additions  *int `json:"additions"`
				Deletions  *int `json:"deletions"`
				PatchBytes *int `json:"patch_bytes"`
			} `json:"totals"`
		} `json:"diff"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}

	diff := decoded.Diff
	if diff.Paths == nil || diff.Files == nil || diff.Kinds == nil || diff.Totals == nil {
		t.Fatalf("diff omits a declared field in %s", encoded)
	}
	if diff.Totals.Files == nil || diff.Totals.Additions == nil ||
		diff.Totals.Deletions == nil || diff.Totals.PatchBytes == nil {
		t.Fatalf("diff.totals omits a declared counter in %s", encoded)
	}
}

func TestZeroDocumentPopulatesEveryClassificationBucket(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}

	var decoded struct {
		Diff struct {
			Kinds map[string]*[]string `json:"kinds"`
		} `json:"diff"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}

	declared := []string{
		"authz", "ci_config", "generated", "iac", "lockfile",
		"migration", "policy_config", "route", "test",
	}
	for _, bucket := range declared {
		paths, present := decoded.Diff.Kinds[bucket]
		if !present || paths == nil {
			t.Errorf("kinds omits %q in %s", bucket, encoded)
			continue
		}
		if len(*paths) != 0 {
			t.Errorf("kinds %q is not empty in %s", bucket, encoded)
		}
	}
	if len(decoded.Diff.Kinds) != len(declared) {
		t.Errorf("kinds carries %d buckets, want %d: %s",
			len(decoded.Diff.Kinds), len(declared), encoded)
	}
}

func TestFileCarriesPathActionAndSizes(t *testing.T) {
	document := domain.Document{Diff: domain.Diff{Files: domain.Declared[domain.File]{{
		Path:       "internal/auth/middleware.go",
		Action:     domain.FileActionModified,
		Additions:  12,
		Deletions:  3,
		PatchBytes: 1840,
	}}}}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}

	want := `"files":[{"path":"internal/auth/middleware.go","action":"modified",` +
		`"additions":12,"deletions":3,"patch_bytes":1840}]`
	if !strings.Contains(string(encoded), want) {
		t.Fatalf("want %s in %s", want, encoded)
	}
}

func TestPathsEmitLexicographicallySorted(t *testing.T) {
	document := domain.Document{Diff: domain.Diff{Paths: domain.Paths{
		"internal/http/routes.go",
		"infra/iam.tf",
		"internal/auth/middleware.go",
	}}}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}

	want := `"paths":["infra/iam.tf","internal/auth/middleware.go","internal/http/routes.go"]`
	if !strings.Contains(string(encoded), want) {
		t.Fatalf("want %s in %s", want, encoded)
	}
}

func TestZeroDocumentEmitsEmptyByDirAsObject(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}
	if !strings.Contains(string(encoded), `"by_dir":{}`) {
		t.Fatalf(`want "by_dir":{} in %s`, encoded)
	}
}

func TestImportEntryDeclaresEveryFieldTheSchemaNames(t *testing.T) {
	document := domain.Document{Diff: domain.Diff{
		Imports: domain.Declared[domain.Import]{
			{Path: "infra/iam.tf", State: domain.ImportStateUnknown},
		},
	}}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}

	for _, declared := range []string{
		`"path":"infra/iam.tf"`, `"state":"unknown"`,
		`"language":null`, `"extractor":null`,
		`"refs":[]`, `"declarations_added":[]`,
	} {
		if !strings.Contains(string(encoded), declared) {
			t.Errorf("want %s in %s", declared, encoded)
		}
	}
}

func TestZeroDocumentDeclaresBothMasksAsLists(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}
	for _, mask := range []string{`"sensitive":[]`, `"unknown":[]`} {
		if !strings.Contains(string(encoded), mask) {
			t.Errorf("want %s in %s", mask, encoded)
		}
	}
}

func TestZeroDocumentDeclaresIaCResourcesAsList(t *testing.T) {
	encoded, err := json.Marshal(domain.Document{})
	if err != nil {
		t.Fatalf("marshal zero document: %v", err)
	}
	if !strings.Contains(string(encoded), `"iac_resources":[]`) {
		t.Fatalf(`want "iac_resources":[] in %s`, encoded)
	}
}
