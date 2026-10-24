// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type ChangeEvidence struct {
	files                     []ChangedFile
	tests                     []changedTest
	testFiles                 []RepositoryPath
	testFilesOfUnknownPackage []RepositoryPath
	packagesEditedBeyondTests []GoImportPath
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
		files: slices.Clone(files), tests: tally.tests, testFiles: tally.testFiles, testFilesOfUnknownPackage: tally.ofUnknownPackage,
		packagesEditedBeyondTests: tally.editedBeyondTests,
	}, nil
}

type testTally struct {
	roots             []ImportRoot
	tests             []changedTest
	testFiles         []RepositoryPath
	ofUnknownPackage  []RepositoryPath
	editedBeyondTests []GoImportPath
}

func (tally *testTally) count(edited EditedTests) error {
	tally.testFiles = append(tally.testFiles, edited.path)
	if !edited.isEditingAnyTest() && !edited.isEditingBeyondTests {
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
	if edited.isEditingBeyondTests {
		tally.editedBeyondTests = append(tally.editedBeyondTests, pkg)
	}
	for _, name := range edited.names {
		tally.tests = append(tally.tests, changedTest{pkg: pkg, name: name})
	}
	return nil
}

func (evidence ChangeEvidence) holdsATestFileOfUnknownPackage() bool {
	return len(evidence.testFilesOfUnknownPackage) > 0
}

func (evidence ChangeEvidence) filesBesideTests() []ChangedFile {
	isTestFile := func(file ChangedFile) bool { return slices.Contains(evidence.testFiles, file.path) }
	return slices.DeleteFunc(slices.Clone(evidence.files), isTestFile)
}

func (edited EditedTests) isEditingAnyTest() bool { return len(edited.names) > 0 }
