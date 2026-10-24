// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
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
	tally := testTally{roots: binding.importRoots()}
	for _, file := range edited {
		if err := tally.count(file); err != nil {
			return ChangeEvidence{}, fmt.Errorf("%s: %w", file.path, err)
		}
	}
	return ChangeEvidence{
		files: slices.Clone(files), tests: tally.tests, testFilesOfUnknownPackage: tally.ofUnknownPackage,
	}, nil
}

type testTally struct {
	roots            []ImportRoot
	tests            []changedTest
	ofUnknownPackage []RepositoryPath
}

func (tally *testTally) count(edited EditedTests) error {
	if !edited.isEditingAnyTest() {
		return nil
	}
	pkg, isNamed, err := testPackageOf(edited.path, tally.roots)
	if err != nil {
		return err
	}
	if !isNamed {
		tally.ofUnknownPackage = append(tally.ofUnknownPackage, edited.path)
		return nil
	}
	for _, name := range edited.names {
		tally.tests = append(tally.tests, changedTest{pkg: pkg, name: name})
	}
	return nil
}

func (edited EditedTests) isEditingAnyTest() bool { return len(edited.names) > 0 }
