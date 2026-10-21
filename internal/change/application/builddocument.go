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
	var imported []domain.ImportedFile
	for _, file := range files {
		if useCase.Extractor == nil || !useCase.Extractor.Claims(file.Path) {
			continue
		}
		head, err := domain.ParseCommitSHA(pullRequest.HeadSHA)
		if err != nil {
			return domain.Document{}, err
		}
		contents, err := useCase.Contents.ContentsAt(ctx, domain.FileAtCommit{Repository: locator.Repository(), Commit: head, Path: file.Path})
		if err != nil {
			return domain.Document{}, err
		}
		source, err := domain.ParseBoundedSource(contents, useCase.Limits)
		if err != nil {
			return domain.Document{}, err
		}
		paths, err := useCase.Extractor.ImportsIn(source)
		if err != nil {
			return domain.Document{}, err
		}
		fileImports, err := domain.ParseFileImports(paths, useCase.Limits)
		if err != nil {
			return domain.Document{}, err
		}
		imported = append(imported, domain.ImportedFile{File: file, Head: fileImports})
	}
	return domain.ParseDocument(pullRequest, domain.Change{Files: bounded, Imported: imported}, useCase.Rules), nil
}
