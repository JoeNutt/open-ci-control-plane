// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type recordedChange struct {
	PullRequest domain.PullRequest
	Tips        domain.BranchTips
	Files       []domain.File
	Stated      []domain.StatedFile
}

type recordedChanges struct {
	Changes map[domain.PullRequestLocator]recordedChange
}

func (source recordedChanges) PullRequestAt(
	_ context.Context, locator domain.PullRequestLocator,
) (domain.ReportedPullRequest, error) {
	change, err := source.changeAt(locator)
	if err != nil {
		return domain.ReportedPullRequest{}, err
	}
	return domain.ReportedPullRequest{PullRequest: change.PullRequest, Tips: change.Tips}, nil
}

func (source recordedChanges) FilesChanged(
	_ context.Context, locator domain.PullRequestLocator,
) ([]domain.StatedFile, error) {
	change, err := source.changeAt(locator)
	if err != nil {
		return nil, err
	}
	stated := make([]domain.StatedFile, 0, len(change.Files)+len(change.Stated))
	for _, file := range change.Files {
		stated = append(stated, statedWithPatchOf(file))
	}
	return append(stated, change.Stated...), nil
}

const onlyContextHunk = "@@ -1,1 +1,1 @@\n "

func statedWithPatchOf(file domain.File) domain.StatedFile {
	if file.PatchBytes == 0 {
		return domain.StatedFile{File: file}
	}
	patch := strings.Repeat("x", file.PatchBytes)
	if file.PatchBytes > len(onlyContextHunk) {
		patch = onlyContextHunk + patch[len(onlyContextHunk):]
	}
	return domain.StatedFile{File: file, Patch: patch, IsPatchStated: true}
}

func (source recordedChanges) changeAt(locator domain.PullRequestLocator) (recordedChange, error) {
	change, isRecorded := source.Changes[locator]
	if !isRecorded {
		return recordedChange{}, fmt.Errorf("%w: %s", domain.ErrNoSuchPullRequest, locator)
	}
	return change, nil
}

type recordedMergeBases struct {
	MergeBases map[domain.BranchTips]domain.CommitSHA
}

func (bases recordedMergeBases) MergeBaseOf(
	_ context.Context, locator domain.PullRequestLocator, tips domain.BranchTips,
) (domain.CommitSHA, error) {
	mergeBase, isRecorded := bases.MergeBases[tips]
	if !isRecorded {
		return domain.CommitSHA{}, fmt.Errorf("%w: %s has no merge base recorded for %s...%s",
			domain.ErrNoMergeBase, locator, tips.Base(), tips.Head())
	}
	return mergeBase, nil
}

type recordedFiles struct {
	Files map[domain.FileAtCommit][]byte
}

func (contents recordedFiles) ContentsAt(_ context.Context, file domain.FileAtCommit) ([]byte, error) {
	answer, isRecorded := contents.Files[file]
	if !isRecorded {
		return nil, fmt.Errorf("%w: %s", domain.ErrNoSuchFile, file)
	}
	return slices.Clone(answer), nil
}

type publishedCommit struct {
	locator domain.PullRequestLocator
	head    domain.CommitSHA
}

type publishedCheckRuns struct {
	published map[publishedCommit]map[string]domain.CheckRun
}

func (runs *publishedCheckRuns) PublishCheckRun(
	_ context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	if runs.published == nil {
		runs.published = map[publishedCommit]map[string]domain.CheckRun{}
	}
	key := publishedCommit{locator: locator, head: head}
	if runs.published[key] == nil {
		runs.published[key] = map[string]domain.CheckRun{}
	}
	runs.published[key][run.Name] = run
	return nil
}

func (runs *publishedCheckRuns) PublishedOn(
	locator domain.PullRequestLocator, head domain.CommitSHA,
) map[string]domain.CheckRun {
	return maps.Clone(runs.published[publishedCommit{locator: locator, head: head}])
}

type dispatchedPlans struct {
	guard      sync.Mutex
	dispatched []domain.TestPlan
}

func (dispatches *dispatchedPlans) DispatchPlan(_ context.Context, plan domain.TestPlan) error {
	dispatches.guard.Lock()
	defer dispatches.guard.Unlock()

	dispatches.dispatched = append(dispatches.dispatched, plan)
	return nil
}

func (dispatches *dispatchedPlans) Dispatched() []domain.TestPlan {
	dispatches.guard.Lock()
	defer dispatches.guard.Unlock()

	return slices.Clone(dispatches.dispatched)
}

type coverageStore struct {
	guard  sync.RWMutex
	held   map[domain.GovernedRepository]domain.RepositoryCoverage
	limits domain.CoverageLimits
}

func newCoverageStore(limits domain.CoverageLimits) *coverageStore {
	return &coverageStore{held: map[domain.GovernedRepository]domain.RepositoryCoverage{}, limits: limits}
}

func (store *coverageStore) ReplaceSuiteCoverage(
	_ context.Context, repository domain.GovernedRepository, suite domain.CoverageSuite, coverage domain.SuiteCoverage,
) error {
	store.guard.Lock()
	defer store.guard.Unlock()

	replaced, err := store.held[repository].Replacing(suite, coverage, store.limits)
	if err != nil {
		return fmt.Errorf("replace coverage of %s in %s: %w", suite, repository, err)
	}
	store.held[repository] = replaced
	return nil
}

func (store *coverageStore) SuiteCoverageOf(
	_ context.Context, repository domain.GovernedRepository, suite domain.CoverageSuite,
) (domain.SuiteCoverage, error) {
	store.guard.RLock()
	defer store.guard.RUnlock()

	return store.held[repository].SuiteCoverageOf(suite)
}

func (store *coverageStore) RepositoryCoverageOf(
	_ context.Context, repository domain.GovernedRepository,
) (domain.RepositoryCoverage, error) {
	store.guard.RLock()
	defer store.guard.RUnlock()

	return store.held[repository], nil
}

type readableHistory struct{}

func (readableHistory) VerifyReadable(context.Context) error { return nil }
