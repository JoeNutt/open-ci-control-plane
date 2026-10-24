// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type headSources map[string]domain.BoundedSource

type headReads struct {
	repository domain.GovernedRepository
	head       domain.CommitSHA
	read       headSources
}

func (read headSources) at(stated statedChange) headReads {
	return headReads{repository: stated.locator.GovernedRepository(), head: stated.tips.Head(), read: read}
}

func (useCase BuildDocument) evidenceUnder(
	ctx context.Context, built BuiltChange, binding domain.Binding,
) (domain.ChangeEvidence, error) {
	files, err := changedFilesOf(built.listed)
	if err != nil {
		return domain.ChangeEvidence{}, fmt.Errorf("read the changed files' lines: %w", err)
	}
	edited, err := useCase.editedTestsOf(ctx, files, built.reads)
	if err != nil {
		return domain.ChangeEvidence{}, fmt.Errorf("read the edited tests: %w", err)
	}
	return domain.ParseChangeEvidence(files, edited, binding)
}

func changedFilesOf(listed []domain.StatedFile) ([]domain.ChangedFile, error) {
	changed := make([]domain.ChangedFile, 0, len(listed))
	for _, stated := range listed {
		parsed, err := domain.ParseChangedFile(stated)
		if err != nil {
			return nil, err
		}
		changed = append(changed, parsed)
	}
	return changed, nil
}

func (useCase BuildDocument) editedTestsOf(
	ctx context.Context, files []domain.ChangedFile, reads headReads,
) ([]domain.EditedTests, error) {
	var edited []domain.EditedTests
	for _, file := range files {
		if !file.ExistsAtHead() || !useCase.Tests.DeclaresTests(file.Path().String()) {
			continue
		}
		functions, err := useCase.testFunctionsAtHead(ctx, file.Path(), reads)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file.Path(), err)
		}
		edited = append(edited, domain.EditedTestsIn(file, functions))
	}
	return edited, nil
}

func (useCase BuildDocument) testFunctionsAtHead(
	ctx context.Context, path domain.RepositoryPath, reads headReads,
) ([]domain.TestFunction, error) {
	source, err := useCase.headSourceOf(ctx, path, reads)
	if err != nil {
		return nil, err
	}
	return useCase.Tests.TestFunctionsIn(source)
}

func (useCase BuildDocument) headSourceOf(
	ctx context.Context, path domain.RepositoryPath, reads headReads,
) (domain.BoundedSource, error) {
	if source, isRead := reads.read[path.String()]; isRead {
		return source, nil
	}
	file, err := domain.ParseFileAtCommit(reads.repository, reads.head, path.String())
	if err != nil {
		return domain.BoundedSource{}, err
	}
	return useCase.sourceAt(ctx, file)
}
