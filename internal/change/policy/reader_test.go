// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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

func TestReadRevisionKeepsThePreviousRevisionThroughARefusedHeadUntilAValidOne(t *testing.T) {
	active := activePolicyLoggingTo(t, slog.DiscardHandler)
	recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
	recordCommit(t, recorded, secondPolicyCommit, liveFilesWithBinding(t, liveBindingPlacingOnePathInTwoGroups(t)))
	recordCommit(t, recorded, thirdPolicyCommit, liveFiles(t))
	reader := readerOver(t, recorded, active)
	readAt := func(head string) error {
		recorded.heads.Heads[readRepository(t)] = policyCommit(t, head)
		return reader.ReadRevision(context.Background())
	}

	if err := readAt(firstPolicyCommit); err != nil {
		t.Fatalf("read the first head: %v", err)
	}
	if err := readAt(secondPolicyCommit); !errors.Is(err, domain.ErrPathInTwoGroups) {
		t.Fatalf("a head placing one path in two groups gave %v, want ErrPathInTwoGroups", err)
	}
	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("after a refused head the revision is %s, want the previous %s", revision, firstPolicyCommit)
	}
	if err := readAt(thirdPolicyCommit); err != nil {
		t.Fatalf("read the third head: %v", err)
	}
	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, thirdPolicyCommit) {
		t.Errorf("after a valid head the revision is %s, want %s", revision, thirdPolicyCommit)
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

func TestReadRevisionReadsEveryPolicyFileFromThePolicyRepositoryAndNoOther(t *testing.T) {
	recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
	var askedAbout []domain.PolicyRepository
	ports := portsOver(recorded)
	ports.Contents = repositoriesAsked{contents: recorded.contents, asked: &askedAbout}
	active := activePolicyLoggingTo(t, slog.DiscardHandler)

	if err := readerWith(t, ports, active).ReadRevision(context.Background()); err != nil {
		t.Fatalf("read revision: %v", err)
	}

	accepted, err := active.Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}
	if workflow := accepted.Model().Bindings()[0].Workflow().String(); workflow != liveWorkflow {
		t.Errorf("the binding used names workflow %q, want the policy repository's %q", workflow, liveWorkflow)
	}
	for _, repository := range askedAbout {
		if repository != readRepository(t) {
			t.Errorf("the reader read a file from %s, want only %s", repository, readPolicyRepository)
		}
	}
	if len(askedAbout) != len(liveFiles(t)) {
		t.Errorf("the reader read %d files, want the %d policy files", len(askedAbout), len(liveFiles(t)))
	}
}

type repositoriesAsked struct {
	contents domain.PolicyContents
	asked    *[]domain.PolicyRepository
}

func (recording repositoriesAsked) ContentsAt(ctx context.Context, at domain.PolicyCommit, path domain.RepositoryPath) ([]byte, error) {
	*recording.asked = append(*recording.asked, at.Repository())
	return recording.contents.ContentsAt(ctx, at, path)
}

func TestReadRevisionReportsARefusedRevisionWithoutItsFileContent(t *testing.T) {
	const contentMarker = "unreported_content_marker"
	reader := readerOver(t,
		policyRepositoryAt(t, firstPolicyCommit, liveFilesWithBinding(t, liveBindingWithUnknownKey(t, contentMarker))),
		activePolicyLoggingTo(t, slog.DiscardHandler))

	err := reader.ReadRevision(context.Background())

	if !errors.Is(err, policy.ErrMalformedPolicyFile) {
		t.Fatalf("a binding with an unknown key gave %v, want ErrMalformedPolicyFile", err)
	}
	if strings.Contains(err.Error(), contentMarker) {
		t.Errorf("the refusal %q carries file content", err)
	}
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

func recordCommit(t *testing.T, recorded recordedPolicyRepository, head string, files []domain.PolicyFile) {
	t.Helper()

	at := policyHeadAt(t, head)
	listed := make([]domain.ListedFile, 0, len(files))
	for _, file := range files {
		listed = append(listed, domain.ListedFile{Path: file.Path, Bytes: len(file.Content)})
		recorded.contents.Files[memory.PolicyFileAt{At: at, Path: repositoryPathOf(t, file.Path)}] = file.Content
	}
	recorded.listings.Listings[at] = listed
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
