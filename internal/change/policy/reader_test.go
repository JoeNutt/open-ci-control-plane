// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"errors"
	"fmt"
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

func TestReadRevisionDoesNotListAgainAHeadItHasAlreadyRead(t *testing.T) {
	recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
	listingsAsked := 0
	ports := portsOver(recorded)
	ports.Listings = countedListings{listings: recorded.listings, asked: &listingsAsked}
	reader := readerWith(t, ports, activePolicyLoggingTo(t, slog.DiscardHandler))

	for range 2 {
		if err := reader.ReadRevision(context.Background()); err != nil {
			t.Fatalf("read revision: %v", err)
		}
	}

	if listingsAsked != 1 {
		t.Errorf("the reader listed the unchanged head %d times, want once", listingsAsked)
	}
}

type countedListings struct {
	listings domain.PolicyListings
	asked    *int
}

func (counted countedListings) ListedFilesAt(ctx context.Context, at domain.PolicyCommit) ([]domain.ListedFile, error) {
	*counted.asked++
	return counted.listings.ListedFilesAt(ctx, at)
}

func TestReadRevisionRefusesAListingOverABoundBeforeFetchingAnyFile(t *testing.T) {
	overBound := map[string]struct {
		listed func([]domain.ListedFile) []domain.ListedFile
		breach error
	}{
		"a file listed one byte over the bound": {
			listed: func(listed []domain.ListedFile) []domain.ListedFile {
				listed[len(listed)-1].Bytes = domain.MaxPolicyFileBytes + 1
				return listed
			},
			breach: domain.ErrPolicyFileTooLarge,
		},
		"one policy file over the count bound": {
			listed: func(listed []domain.ListedFile) []domain.ListedFile {
				for index := len(listed); index <= domain.MaxPolicyFiles+1; index++ {
					listed = append(listed, domain.ListedFile{Path: fmt.Sprintf(extraBindingPathFormat, index), Bytes: 1})
				}
				return listed
			},
			breach: domain.ErrTooManyPolicyFiles,
		},
	}
	for form, bound := range overBound {
		t.Run(form, func(t *testing.T) {
			recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
			at := policyHeadAt(t, firstPolicyCommit)
			recorded.listings.Listings[at] = bound.listed(recorded.listings.Listings[at])
			contentsAsked := 0
			ports := portsOver(recorded)
			ports.Contents = countedContents{contents: recorded.contents, asked: &contentsAsked}

			err := readerWith(t, ports, activePolicyLoggingTo(t, slog.DiscardHandler)).ReadRevision(context.Background())

			if !errors.Is(err, bound.breach) {
				t.Fatalf("%s gave %v, want %v", form, err, bound.breach)
			}
			if contentsAsked != 0 {
				t.Errorf("the reader fetched %d files from a listing over the bound", contentsAsked)
			}
		})
	}
}

type countedContents struct {
	contents domain.PolicyContents
	asked    *int
}

func (counted countedContents) ContentsAt(ctx context.Context, at domain.PolicyCommit, path domain.RepositoryPath) ([]byte, error) {
	*counted.asked++
	return counted.contents.ContentsAt(ctx, at, path)
}

func TestParseReaderRefusesAReaderMissingAPart(t *testing.T) {
	missing := map[string]func(*readerParts){
		"no repository":    func(parts *readerParts) { parts.repository = domain.PolicyRepository{} },
		"no heads":         func(parts *readerParts) { parts.ports.Heads = nil },
		"no listings":      func(parts *readerParts) { parts.ports.Listings = nil },
		"no contents":      func(parts *readerParts) { parts.ports.Contents = nil },
		"no active policy": func(parts *readerParts) { parts.active = nil },
	}
	for form, remove := range missing {
		t.Run(form, func(t *testing.T) {
			parts := completeReaderParts(t)
			remove(&parts)

			if _, err := policy.ParseReader(parts.repository, parts.ports, parts.active); !errors.Is(err, policy.ErrIncompleteReader) {
				t.Fatalf("a reader with %s gave %v, want ErrIncompleteReader", form, err)
			}
		})
	}
}

type readerParts struct {
	repository domain.PolicyRepository
	ports      policy.ReaderPorts
	active     *policy.ActivePolicy
}

func completeReaderParts(t *testing.T) readerParts {
	t.Helper()

	return readerParts{
		repository: readRepository(t),
		ports:      portsOver(policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))),
		active:     activePolicyLoggingTo(t, slog.DiscardHandler),
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
	at := policyHeadAt(t, head)
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

func policyHeadAt(t *testing.T, head string) domain.PolicyCommit {
	t.Helper()

	at, err := domain.ParsePolicyCommit(readRepository(t), policyCommit(t, head))
	if err != nil {
		t.Fatalf("parse policy commit: %v", err)
	}
	return at
}

func readerOver(t *testing.T, recorded recordedPolicyRepository, active *policy.ActivePolicy) *policy.Reader {
	t.Helper()

	return readerWith(t, portsOver(recorded), active)
}

func readerWith(t *testing.T, ports policy.ReaderPorts, active *policy.ActivePolicy) *policy.Reader {
	t.Helper()

	reader, err := policy.ParseReader(readRepository(t), ports, active)
	if err != nil {
		t.Fatalf("parse reader: %v", err)
	}
	return reader
}

func portsOver(recorded recordedPolicyRepository) policy.ReaderPorts {
	return policy.ReaderPorts{Heads: recorded.heads, Listings: recorded.listings, Contents: recorded.contents}
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
