// SPDX-License-Identifier: GPL-3.0-or-later

// Package application assembles a change document from what a forge reports.
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
	imported, err := useCase.importedFilesOf(
		ctx, locator, pullRequest.HeadSHA, useCase.readableAtHeadOf(files))
	if err != nil {
		return domain.Document{}, fmt.Errorf("extract the imports of %s: %w", locator, err)
	}
	change := domain.Change{Files: bounded, Imported: imported}
	return domain.ParseDocument(pullRequest, change, useCase.Rules), nil
}

func (useCase BuildDocument) readableAtHeadOf(files []domain.File) []domain.File {
	var readable []domain.File
	for _, file := range files {
		if useCase.Extractor.Claims(file.Path) && file.Action.ExistsAtHead() {
			readable = append(readable, file)
		}
	}
	return readable
}

func (useCase BuildDocument) importedFilesOf(
	ctx context.Context, locator domain.PullRequestLocator, headSHA string, readable []domain.File,
) ([]domain.ImportedFile, error) {
	imported := make([]domain.ImportedFile, 0, len(readable))
	for _, file := range readable {
		head, err := useCase.importsAtHeadOf(ctx, locator, headSHA, file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file.Path, err)
		}
		var base domain.FileImports
		if useCase.Bases != nil && file.Action.ExistsAtBase() {
			baseCommit, err := useCase.Bases.BaseCommitOf(ctx, locator)
			if err != nil {
				return nil, err
			}
			base, err = useCase.importsAt(ctx, domain.FileAtCommit{Repository: locator.Repository(), Commit: baseCommit, Path: file.Path})
			if err != nil {
				return nil, err
			}
		}
		imported = append(imported, domain.ImportedFile{File: file, Base: base, Head: head})
	}
	return imported, nil
}

func (useCase BuildDocument) importsAtHeadOf(
	ctx context.Context, locator domain.PullRequestLocator, headSHA string, file domain.File,
) (domain.FileImports, error) {
	head, err := domain.ParseCommitSHA(headSHA)
	if err != nil {
		return domain.FileImports{}, fmt.Errorf("the head commit: %w", err)
	}
	return useCase.importsAt(ctx, domain.FileAtCommit{
		Repository: locator.Repository(), Commit: head, Path: file.Path,
	})
}

func (useCase BuildDocument) importsAt(
	ctx context.Context, file domain.FileAtCommit,
) (domain.FileImports, error) {
	contents, err := useCase.Contents.ContentsAt(ctx, file)
	if err != nil {
		return domain.FileImports{}, fmt.Errorf("read at %s: %w", file.Commit, err)
	}
	source, err := domain.ParseBoundedSource(contents, useCase.Limits)
	if err != nil {
		return domain.FileImports{}, fmt.Errorf("at %s: %w", file.Commit, err)
	}
	paths, err := useCase.Extractor.ImportsIn(source)
	if err != nil {
		return domain.FileImports{}, fmt.Errorf("at %s: %w", file.Commit, err)
	}
	return domain.ParseFileImports(paths, useCase.Limits)
}
