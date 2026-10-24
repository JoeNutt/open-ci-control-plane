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
	Tests     domain.TestExtractor
	Limits    domain.Limits
	Rules     domain.ClassificationRules
}

type BuiltChange struct {
	Document domain.Document
	listed   []domain.StatedFile
	reads    headReads
}

func (useCase BuildDocument) Execute(
	ctx context.Context, locator domain.PullRequestLocator,
) (BuiltChange, error) {
	reported, err := useCase.Source.PullRequestAt(ctx, locator)
	if err != nil {
		return BuiltChange{}, fmt.Errorf("read pull request %s: %w", locator, err)
	}
	listed, err := useCase.Source.FilesChanged(ctx, locator)
	if err != nil {
		return BuiltChange{}, fmt.Errorf("read changed files of %s: %w", locator, err)
	}
	files := documentFilesOf(listed)
	bounded, err := domain.ParseBoundedFiles(files, useCase.Limits)
	if err != nil {
		return BuiltChange{}, fmt.Errorf("bound %s: %w", locator, overBoundAt(reported.Tips, err))
	}
	stated := statedChange{locator: locator, tips: reported.Tips}
	imported, read, err := useCase.importedFilesOf(ctx, stated, useCase.claimedOf(files))
	if err != nil {
		return BuiltChange{}, fmt.Errorf("extract the imports of %s: %w", locator, err)
	}
	facts, err := domain.ParseStructuralFacts(imported, useCase.Limits)
	if err != nil {
		return BuiltChange{}, fmt.Errorf("bound the structural facts of %s: %w", locator, err)
	}
	change := domain.Change{Files: bounded, Structure: facts}
	document, err := useCase.boundedDocumentOf(reported, change)
	if err != nil {
		return BuiltChange{}, fmt.Errorf("bound the document of %s: %w", locator, err)
	}
	return BuiltChange{Document: document, listed: listed, reads: read.at(stated)}, nil
}

func documentFilesOf(listed []domain.StatedFile) []domain.File {
	files := make([]domain.File, 0, len(listed))
	for _, stated := range listed {
		files = append(files, stated.File)
	}
	return files
}

func (useCase BuildDocument) boundedDocumentOf(
	reported domain.ReportedPullRequest, change domain.Change,
) (domain.Document, error) {
	document, err := domain.ParseBoundedDocument(domain.ParseDocument(reported.PullRequest, change, useCase.Rules), useCase.Limits)
	if err != nil {
		return domain.Document{}, overBoundAt(reported.Tips, err)
	}
	return document, nil
}

func overBoundAt(tips domain.BranchTips, breach error) domain.ChangeOverBound {
	return domain.ChangeOverBound{Head: tips.Head(), Breach: breach}
}

type statedChange struct {
	locator domain.PullRequestLocator
	tips    domain.BranchTips
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
) ([]domain.ImportedFile, headSources, error) {
	read := headSources{}
	if len(claimed) == 0 {
		return nil, read, nil
	}
	ends, err := useCase.endsOf(ctx, stated)
	if err != nil {
		return nil, nil, err
	}
	imported := make([]domain.ImportedFile, 0, len(claimed))
	for _, file := range claimed {
		importedFile, err := useCase.importedFileAt(ctx, ends, file, read)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", file.Path, err)
		}
		imported = append(imported, importedFile)
	}
	return imported, read, nil
}

func (useCase BuildDocument) endsOf(ctx context.Context, stated statedChange) (changeEnds, error) {
	base, err := useCase.Bases.MergeBaseOf(ctx, stated.locator, stated.tips)
	if err != nil {
		return changeEnds{}, fmt.Errorf("the merge base: %w", err)
	}
	return changeEnds{repository: stated.locator.GovernedRepository(), base: base, head: stated.tips.Head()}, nil
}

func (useCase BuildDocument) importedFileAt(
	ctx context.Context, ends changeEnds, file domain.File, read headSources,
) (domain.ImportedFile, error) {
	imported := domain.ImportedFile{File: file}
	if file.Action.ExistsAtBase() {
		base, err := useCase.importsAt(ctx, ends.repository, ends.base, file.Path)
		if err != nil {
			return domain.ImportedFile{}, fmt.Errorf("at base: %w", err)
		}
		imported.Base = base.imports
	}
	if file.Action.ExistsAtHead() {
		head, err := useCase.importsAt(ctx, ends.repository, ends.head, file.Path)
		if err != nil {
			return domain.ImportedFile{}, fmt.Errorf("at head: %w", err)
		}
		imported.Head = head.imports
		read[file.Path] = head.source
	}
	return imported, nil
}

type sourceImports struct {
	source  domain.BoundedSource
	imports domain.FileImports
}

func (useCase BuildDocument) importsAt(
	ctx context.Context, repository domain.GovernedRepository, commit domain.CommitSHA, path string,
) (sourceImports, error) {
	file, err := domain.ParseFileAtCommit(repository, commit, path)
	if err != nil {
		return sourceImports{}, err
	}
	source, err := useCase.sourceAt(ctx, file)
	if err != nil {
		return sourceImports{}, err
	}
	paths, err := useCase.Extractor.ImportsIn(source)
	if err != nil {
		return sourceImports{}, fmt.Errorf("at %s: %w", file.Commit(), err)
	}
	imports, err := domain.ParseFileImports(paths, useCase.Limits)
	if err != nil {
		return sourceImports{}, err
	}
	return sourceImports{source: source, imports: imports}, nil
}

func (useCase BuildDocument) sourceAt(ctx context.Context, file domain.FileAtCommit) (domain.BoundedSource, error) {
	contents, err := useCase.Contents.ContentsAt(ctx, file)
	if err != nil {
		return domain.BoundedSource{}, err
	}
	source, err := domain.ParseBoundedSource(contents, useCase.Limits)
	if err != nil {
		return domain.BoundedSource{}, fmt.Errorf("at %s: %w", file.Commit(), err)
	}
	return source, nil
}
