// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

type ChangeEvidence struct {
	files                     []ChangedFile
	tests                     []changedTest
	testFilesOfUnknownPackage []RepositoryPath
}

type changedTest struct {
	pkg  GoImportPath
	name TopLevelTestName
}

func ParseChangeEvidence(files []ChangedFile, edited []EditedTests, binding Binding) (ChangeEvidence, error) {
	roots := binding.importRoots()
	var tests []changedTest
	var unknown []RepositoryPath
	for _, file := range edited {
		named, err := file.changedTestsUnder(roots)
		if errors.Is(err, errNoImportRootHoldsTheTest) {
			unknown = append(unknown, file.path)
			continue
		}
		if err != nil {
			return ChangeEvidence{}, fmt.Errorf("%s: %w", file.path, err)
		}
		tests = append(tests, named...)
	}
	return ChangeEvidence{files: slices.Clone(files), tests: tests, testFilesOfUnknownPackage: unknown}, nil
}

func (edited EditedTests) changedTestsUnder(roots []ImportRoot) ([]changedTest, error) {
	if !edited.isEditingAnyTest() {
		return nil, nil
	}
	pkg, err := testPackageOf(edited.path, roots)
	if err != nil {
		return nil, err
	}
	tests := make([]changedTest, 0, len(edited.names))
	for _, name := range edited.names {
		tests = append(tests, changedTest{pkg: pkg, name: name})
	}
	return tests, nil
}

func (edited EditedTests) isEditingAnyTest() bool { return len(edited.names) > 0 }
