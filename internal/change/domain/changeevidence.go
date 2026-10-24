// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type ChangeEvidence struct {
	files []ChangedFile
	tests []ChangedTest
}

type ChangedTest struct {
	pkg  GoImportPath
	name TopLevelTestName
}

func ParseChangeEvidence(files []ChangedFile, edited []EditedTests, binding Binding) (ChangeEvidence, error) {
	roots := binding.importRoots()
	var tests []ChangedTest
	for _, file := range edited {
		named, err := file.changedTestsUnder(roots)
		if err != nil {
			return ChangeEvidence{}, fmt.Errorf("%s: %w", file.path, err)
		}
		tests = append(tests, named...)
	}
	return ChangeEvidence{files: slices.Clone(files), tests: tests}, nil
}

func (edited EditedTests) changedTestsUnder(roots []ImportRoot) ([]ChangedTest, error) {
	if !edited.isEditingAnyTest() {
		return nil, nil
	}
	pkg, err := testPackageOf(edited.path, roots)
	if err != nil {
		return nil, err
	}
	tests := make([]ChangedTest, 0, len(edited.names))
	for _, name := range edited.names {
		tests = append(tests, ChangedTest{pkg: pkg, name: name})
	}
	return tests, nil
}

func (edited EditedTests) isEditingAnyTest() bool { return len(edited.names) > 0 }
