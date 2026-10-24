// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ChangeEvidence struct {
	files []ChangedFile
}

func ParseChangeEvidence(files []ChangedFile, _ []EditedTests, _ Binding) (ChangeEvidence, error) {
	return ChangeEvidence{files: slices.Clone(files)}, nil
}
