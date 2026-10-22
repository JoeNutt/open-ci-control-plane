// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type BuildDocument struct {
	Source    domain.ChangeSource
	Bases     domain.ChangeBase
	Contents  domain.FileContents
	Extractor domain.ImportExtractor
	Limits    domain.Limits
	Rules     domain.ClassificationRules
}

func (useCase BuildDocument) Execute(
	ctx context.Context, locator domain.PullRequestLocator,
) (domain.Document, error) {
	pullRequest, err := useCase.Source.PullRequestAt(ctx, locator)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read pull request %s: %w", locator, err)
	}
	files, err := useCase.Source.FilesChanged(ctx, locator)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read changed files of %s: %w", locator, err)
	}
	bounded, err := domain.ParseBoundedFiles(files, useCase.Limits)
	if err != nil {
		return domain.Document{}, fmt.Errorf("bound %s: %w", locator, err)
	}
	stated := statedChange{locator: locator, headSHA: pullRequest.HeadSHA, baseSHA: pullRequest.BaseSHA}
	imported, err := useCase.importedFilesOf(ctx, stated, useCase.claimedOf(files))
	if err != nil {
		return domain.Document{}, fmt.Errorf("extract the imports of %s: %w", locator, err)
	}
	facts, err := domain.ParseStructuralFacts(imported, useCase.Limits)
	if err != nil {
		return domain.Document{}, fmt.Errorf("bound the structural facts of %s: %w", locator, err)
	}
	change := domain.Change{Files: bounded, Structure: facts}
	return domain.ParseDocument(pullRequest, change, useCase.Rules), nil
}

type statedChange struct {
	locator domain.PullRequestLocator
	headSHA string
	baseSHA string
}

type changeEnds struct {
	repository domain.GovernedRepository
	base       domain.CommitSHA
	head       domain.CommitSHA
}

func (useCase BuildDocument) claimedOf(files []domain.File) []domain.File {
	var claimed []domain.File
	for _, file := range files {
		if useCase.Extractor.Claims(file.Path) {
			claimed = append(claimed, file)
		}
	}
	return claimed
}

func (useCase BuildDocument) importedFilesOf(
	ctx context.Context, stated statedChange, claimed []domain.File,
) ([]domain.ImportedFile, error) {
	if len(claimed) == 0 {
		return nil, nil
	}
	ends, err := useCase.endsOf(ctx, stated)
	if err != nil {
		return nil, err
	}
	imported := make([]domain.ImportedFile, 0, len(claimed))
	for _, file := range claimed {
		importedFile, err := useCase.importedFileAt(ctx, ends, file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file.Path, err)
		}
		imported = append(imported, importedFile)
	}
	return imported, nil
}

func (useCase BuildDocument) endsOf(ctx context.Context, stated statedChange) (changeEnds, error) {
	head, err := domain.ParseCommitSHA(stated.headSHA)
	if err != nil {
		return changeEnds{}, fmt.Errorf("the head commit: %w", err)
	}
	baseTip, err := domain.ParseCommitSHA(stated.baseSHA)
	if err != nil {
		return changeEnds{}, fmt.Errorf("the base commit: %w", err)
	}
	base, err := useCase.Bases.MergeBaseOf(ctx, stated.locator, domain.BranchTips{Base: baseTip, Head: head})
	if err != nil {
		return changeEnds{}, fmt.Errorf("the merge base: %w", err)
	}
	return changeEnds{repository: stated.locator.GovernedRepository(), base: base, head: head}, nil
}

func (useCase BuildDocument) importedFileAt(
	ctx context.Context, ends changeEnds, file domain.File,
) (domain.ImportedFile, error) {
	imported := domain.ImportedFile{File: file}
	var err error
	if file.Action.ExistsAtBase() {
		imported.Base, err = useCase.importsAt(ctx, ends, ends.base, file.Path)
		if err != nil {
			return domain.ImportedFile{}, fmt.Errorf("at base: %w", err)
		}
	}
	if file.Action.ExistsAtHead() {
		imported.Head, err = useCase.importsAt(ctx, ends, ends.head, file.Path)
		if err != nil {
			return domain.ImportedFile{}, fmt.Errorf("at head: %w", err)
		}
	}
	return imported, nil
}

func (useCase BuildDocument) importsAt(
	ctx context.Context, ends changeEnds, commit domain.CommitSHA, path string,
) (domain.FileImports, error) {
	file, err := domain.ParseFileAtCommit(ends.repository, commit, path)
	if err != nil {
		return domain.FileImports{}, err
	}
	contents, err := useCase.Contents.ContentsAt(ctx, file)
	if err != nil {
		return domain.FileImports{}, err
	}
	source, err := domain.ParseBoundedSource(contents, useCase.Limits)
	if err != nil {
		return domain.FileImports{}, fmt.Errorf("at %s: %w", file.Commit(), err)
	}
	paths, err := useCase.Extractor.ImportsIn(source)
	if err != nil {
		return domain.FileImports{}, fmt.Errorf("at %s: %w", file.Commit(), err)
	}
	return domain.ParseFileImports(paths, useCase.Limits)
}
