// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const readPolicyRepository = "JoeNutt/open-ci-policy"

func TestReadRevisionAcceptsThePolicyFilesAtTheDefaultBranchHeadAsThatRevision(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	reader := readerOver(t, policyRepositoryAt(t, firstPolicyCommit, liveFiles(t)), active)

	if err := reader.ReadRevision(context.Background()); err != nil {
		t.Fatalf("read revision: %v", err)
	}
	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("accepted revision %s, want the head %s", revision, firstPolicyCommit)
	}
}

func TestParseReaderRefusesAReaderMissingAPart(t *testing.T) {
	recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
	complete := policy.ReaderPorts{Heads: recorded.heads, Listings: recorded.listings, Contents: recorded.contents}
	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	missing := map[string]func() (domain.PolicyRepository, policy.ReaderPorts, *policy.ActivePolicy){
		"no repository": func() (domain.PolicyRepository, policy.ReaderPorts, *policy.ActivePolicy) {
			return domain.PolicyRepository{}, complete, active
		},
		"no heads": func() (domain.PolicyRepository, policy.ReaderPorts, *policy.ActivePolicy) {
			ports := complete
			ports.Heads = nil
			return readRepository(t), ports, active
		},
		"no listings": func() (domain.PolicyRepository, policy.ReaderPorts, *policy.ActivePolicy) {
			ports := complete
			ports.Listings = nil
			return readRepository(t), ports, active
		},
		"no contents": func() (domain.PolicyRepository, policy.ReaderPorts, *policy.ActivePolicy) {
			ports := complete
			ports.Contents = nil
			return readRepository(t), ports, active
		},
		"no active policy": func() (domain.PolicyRepository, policy.ReaderPorts, *policy.ActivePolicy) {
			return readRepository(t), complete, nil
		},
	}
	for form, parts := range missing {
		t.Run(form, func(t *testing.T) {
			if _, err := policy.ParseReader(parts()); !errors.Is(err, policy.ErrIncompleteReader) {
				t.Fatalf("a reader with %s gave %v, want ErrIncompleteReader", form, err)
			}
		})
	}
}

type recordedPolicyRepository struct {
	heads    memory.DefaultBranchHeads
	listings memory.PolicyListings
	contents memory.PolicyContents
}

func policyRepositoryAt(t *testing.T, head string, files []domain.PolicyFile) recordedPolicyRepository {
	t.Helper()

	repository := readRepository(t)
	at, err := domain.ParsePolicyCommit(repository, policyCommit(t, head))
	if err != nil {
		t.Fatalf("parse policy commit: %v", err)
	}
	listed := []domain.ListedFile{{Path: "README.md", Bytes: len("# policy\n")}}
	recorded := map[memory.PolicyFileAt][]byte{}
	for _, file := range files {
		listed = append(listed, domain.ListedFile{Path: file.Path, Bytes: len(file.Content)})
		recorded[memory.PolicyFileAt{At: at, Path: repositoryPathOf(t, file.Path)}] = file.Content
	}
	return recordedPolicyRepository{
		heads:    memory.DefaultBranchHeads{Heads: map[domain.PolicyRepository]domain.CommitSHA{repository: at.Commit()}},
		listings: memory.PolicyListings{Listings: map[domain.PolicyCommit][]domain.ListedFile{at: listed}},
		contents: memory.PolicyContents{Files: recorded},
	}
}

func readerOver(t *testing.T, recorded recordedPolicyRepository, active *policy.ActivePolicy) *policy.Reader {
	t.Helper()

	reader, err := policy.ParseReader(readRepository(t), policy.ReaderPorts{
		Heads:    recorded.heads,
		Listings: recorded.listings,
		Contents: recorded.contents,
	}, active)
	if err != nil {
		t.Fatalf("parse reader: %v", err)
	}
	return reader
}

func readRepository(t *testing.T) domain.PolicyRepository {
	t.Helper()

	repository, err := domain.ParsePolicyRepository(readPolicyRepository)
	if err != nil {
		t.Fatalf("parse policy repository: %v", err)
	}
	return repository
}

func repositoryPathOf(t *testing.T, stated string) domain.RepositoryPath {
	t.Helper()

	path, err := domain.ParseRepositoryPath(stated)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", stated, err)
	}
	return path
}
