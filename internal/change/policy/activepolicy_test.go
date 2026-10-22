// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const (
	firstPolicyCommit  = "1111111111111111111111111111111111111111"
	secondPolicyCommit = "2222222222222222222222222222222222222222"
)

func TestParseActivePolicyRefusesANilLogger(t *testing.T) {
	if _, err := policy.ParseActivePolicy(nil); !errors.Is(err, policy.ErrNoPolicyLog) {
		t.Fatalf("a nil logger gave %v, want ErrNoPolicyLog", err)
	}
}

func TestActivePolicyHoldsNoRevisionBeforeOneIsAccepted(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)

	if _, err := active.Accepted(); !errors.Is(err, policy.ErrNoPolicyLoaded) {
		t.Fatalf("a fresh active policy gave %v, want ErrNoPolicyLoaded", err)
	}
}

func TestActivePolicyStaysUnloadedWhenTheFirstCandidateIsRefused(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)

	if err := active.Offer(policyCommit(t, firstPolicyCommit), liveFilesWithBinding(t, liveBindingWithAlias(t))); !errors.Is(err, policy.ErrAliasOrAnchor) {
		t.Fatalf("a first candidate with a YAML alias gave %v, want ErrAliasOrAnchor", err)
	}
	if _, err := active.Accepted(); !errors.Is(err, policy.ErrNoPolicyLoaded) {
		t.Fatalf("after a refused first candidate the active policy gave %v, want ErrNoPolicyLoaded", err)
	}
}

func TestActivePolicyAcceptsAValidCandidateAtItsCommit(t *testing.T) {
	active := activeAtFirstCommit(t)

	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("accepted revision %s, want %s", revision, firstPolicyCommit)
	}
}

func TestActivePolicyServesTheModelParsedAtTheAcceptedCommit(t *testing.T) {
	accepted, err := activeAtFirstCommit(t).Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}

	bindings := accepted.Model().Bindings()
	if len(bindings) != 1 || bindings[0].Repository().String() != liveRepository {
		t.Errorf("accepted model binds %v, want the live binding for %s", bindings, liveRepository)
	}
}

func TestActivePolicyKeepsThePreviousRevisionForEveryRefusalReason(t *testing.T) {
	refused := map[string]struct {
		candidate []domain.PolicyFile
		reason    error
	}{
		"an unknown key":                     {liveFilesWithBinding(t, liveBindingWithUnknownKey(t, "timeout")), policy.ErrMalformedPolicyFile},
		"a YAML alias":                       {liveFilesWithBinding(t, liveBindingWithAlias(t)), policy.ErrAliasOrAnchor},
		"a YAML anchor":                      {liveFilesWithCatalog(t, liveCatalogWithAnchor(t)), policy.ErrAliasOrAnchor},
		"two bindings for one repository":    {liveFilesPlus(t, "bindings/JoeNutt/copy.yaml", liveBindingPath), domain.ErrDuplicateBinding},
		"two catalog policies with one name": {liveFilesPlus(t, "catalog/copy.yaml", liveCatalogPath), domain.ErrDuplicateCatalogPolicy},
		"a binding naming an unknown policy": {liveFilesWithBinding(t, liveBindingWith(t, "use: clean-architecture", "use: mvvm")), domain.ErrUnknownCatalogPolicy},
		"a binding naming an undeclared group": {
			liveFilesWithBinding(t, liveBindingWith(t, liveDriverPaths, liveDriverPaths+"\n      infrastructure: [\"deploy/**\"]")),
			domain.ErrUndeclaredGroup,
		},
		"a missing workflow":             {liveFilesWithBinding(t, liveBindingWithoutWorkflow(t)), domain.ErrMissingWorkflow},
		"an empty import-root prefix":    {liveFilesWithBinding(t, liveBindingWith(t, "prefix: "+liveModule, `prefix: ""`)), domain.ErrMalformedGoImportPath},
		"two roots sharing one prefix":   {liveFilesWithBinding(t, liveBindingWithImportRoots(t, 1)), domain.ErrSharedImportRootPrefix},
		"one path in two groups":         {liveFilesWithBinding(t, liveBindingPlacingOnePathInTwoGroups(t)), domain.ErrPathInTwoGroups},
		"a file outside the directories": {liveFilesPlus(t, "templates/copy.yaml", liveBindingPath), policy.ErrUnplacedPolicyFile},
		"a file above 1 MiB": {
			liveFilesWithBinding(t, string(fixture(t, liveBindingPath))+strings.Repeat(" ", domain.MaxPolicyFileBytes)),
			domain.ErrPolicyFileTooLarge,
		},
		"more than 256 files":             {policyFilesBeyondTheBound(t), domain.ErrTooManyPolicyFiles},
		"more than 64 module groups":      {liveFilesWithBinding(t, liveBindingWithGroups(t, domain.MaxBindingGroups)), domain.ErrTooManyBindingGroups},
		"more than 512 path patterns":     {liveFilesWithBinding(t, liveBindingWithDriverPatterns(t, domain.MaxBindingPathPatterns)), domain.ErrTooManyBindingPathPatterns},
		"more than 64 import roots":       {liveFilesWithBinding(t, liveBindingWithImportRoots(t, domain.MaxBindingImportRoots)), domain.ErrTooManyBindingImportRoots},
		"more than 512 always-run suites": {liveFilesWithBinding(t, liveBindingWithAlwaysRun(t, domain.MaxBindingAlwaysRunSuites)), domain.ErrTooManyAlwaysRunSuites},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			active := activeAtFirstCommit(t)

			if err := active.Offer(policyCommit(t, secondPolicyCommit), refusal.candidate); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
			if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
				t.Errorf("accepted revision %s after %s, want the previous %s", revision, form, firstPolicyCommit)
			}
		})
	}
}

func TestActivePolicyLogsTheRevisionItAccepts(t *testing.T) {
	var logged bytes.Buffer
	active := activePolicyLoggingTo(t, slog.NewJSONHandler(&logged, nil))

	if err := active.Offer(policyCommit(t, firstPolicyCommit), liveFiles(t)); err != nil {
		t.Fatalf("offer the live policy files: %v", err)
	}
	if !strings.Contains(logged.String(), firstPolicyCommit) {
		t.Errorf("acceptance log %q does not name %s", logged.String(), firstPolicyCommit)
	}
}

func TestActivePolicyLogsARefusalWithItsReasonAndNoFileContent(t *testing.T) {
	var logged bytes.Buffer
	active := activePolicyLoggingTo(t, slog.NewJSONHandler(&logged, nil))
	const contentMarker = "unlogged_content_marker"
	candidate := liveFiles(t)
	candidate[1].Content = []byte(liveBindingWithUnknownKey(t, contentMarker))

	if err := active.Offer(policyCommit(t, secondPolicyCommit), candidate); !errors.Is(err, policy.ErrMalformedPolicyFile) {
		t.Fatalf("a binding with an unknown key gave %v, want ErrMalformedPolicyFile", err)
	}
	for _, wanted := range []string{secondPolicyCommit, policy.ErrMalformedPolicyFile.Error()} {
		if !strings.Contains(logged.String(), wanted) {
			t.Errorf("refusal log %q does not name %q", logged.String(), wanted)
		}
	}
	if strings.Contains(logged.String(), contentMarker) {
		t.Errorf("refusal log %q carries file content", logged.String())
	}
}

func activePolicyLoggingTo(t *testing.T, handler slog.Handler) *policy.ActivePolicy {
	t.Helper()

	active, err := policy.ParseActivePolicy(slog.New(handler))
	if err != nil {
		t.Fatalf("parse active policy: %v", err)
	}
	return active
}

func activeAtFirstCommit(t *testing.T) *policy.ActivePolicy {
	t.Helper()

	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	if err := active.Offer(policyCommit(t, firstPolicyCommit), liveFiles(t)); err != nil {
		t.Fatalf("offer the live policy files: %v", err)
	}
	return active
}

func acceptedRevisionOf(t *testing.T, active *policy.ActivePolicy) domain.CommitSHA {
	t.Helper()

	accepted, err := active.Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}
	return accepted.Revision()
}

func policyCommit(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %s: %v", stated, err)
	}
	return commit
}

func liveFilesWithBinding(t *testing.T, binding string) []domain.PolicyFile {
	t.Helper()

	files := liveFiles(t)
	files[1].Content = []byte(binding)
	return files
}

func liveFilesWithCatalog(t *testing.T, catalog string) []domain.PolicyFile {
	t.Helper()

	files := liveFiles(t)
	files[0].Content = []byte(catalog)
	return files
}

func liveFilesPlus(t *testing.T, path, copiedFixture string) []domain.PolicyFile {
	t.Helper()

	return append(liveFiles(t), domain.PolicyFile{Path: path, Content: fixture(t, copiedFixture)})
}

func policyFilesBeyondTheBound(t *testing.T) []domain.PolicyFile {
	t.Helper()

	files := liveFiles(t)
	for index := len(files); index <= domain.MaxPolicyFiles; index++ {
		files = append(files, domain.PolicyFile{Path: fmt.Sprintf("bindings/owner/repository%d.yaml", index), Content: files[1].Content})
	}
	return files
}

func liveBindingWithGroups(t *testing.T, beyondTheLive int) string {
	t.Helper()

	return liveBindingWith(t, liveDriverPaths, liveDriverPaths+linesOf(beyondTheLive, "\n      group%d: []"))
}

func liveBindingWithDriverPatterns(t *testing.T, beyondTheLive int) string {
	t.Helper()

	return liveBindingWith(t, liveDriverPaths, `driver: ["cmd/**"`+linesOf(beyondTheLive, `, "cmd%d/**"`)+"]")
}

func liveBindingWithImportRoots(t *testing.T, beyondTheLive int) string {
	t.Helper()

	return liveBindingWith(t, liveRootPathLine+"\n", liveRootPathLine+"\n"+linesOf(beyondTheLive, "      - prefix: "+liveModule+"\n        path: internal%d\n"))
}

func liveBindingWithAlwaysRun(t *testing.T, beyondTheLive int) string {
	t.Helper()

	liveSuite := "    - " + liveModule + "/internal/change/domain"

	return liveBindingWith(t, liveSuite, liveSuite+linesOf(beyondTheLive, "\n    - "+liveModule+"/internal/suite%d"))
}

func linesOf(count int, format string) string {
	var lines strings.Builder
	for index := range count {
		fmt.Fprintf(&lines, format, index)
	}
	return lines.String()
}
