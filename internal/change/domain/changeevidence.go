// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ChangeEvidence struct {
	files []ChangedFile
}

func ParseChangeEvidence(files []ChangedFile, edited []EditedTests, binding Binding) (ChangeEvidence, error) {
	return ChangeEvidence{files: files}, nil
}
