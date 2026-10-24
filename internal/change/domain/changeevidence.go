// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ChangeEvidence struct {
	files []ChangedFile
	tests []ChangedTest
}

type ChangedTest struct {
	pkg  GoImportPath
	name TopLevelTestName
}

func ParseChangeEvidence(files []ChangedFile, edited []EditedTests, binding Binding) (ChangeEvidence, error) {
	var roots []ImportRoot
	for _, policy := range binding.policies {
		roots = append(roots, policy.importRoots...)
	}
	var tests []ChangedTest
	for _, file := range edited {
		pkg, err := testPackageOf(file.path, roots)
		if err != nil {
			return ChangeEvidence{}, err
		}
		for _, name := range file.names {
			tests = append(tests, ChangedTest{pkg: pkg, name: name})
		}
	}
	return ChangeEvidence{files: slices.Clone(files), tests: tests}, nil
}
