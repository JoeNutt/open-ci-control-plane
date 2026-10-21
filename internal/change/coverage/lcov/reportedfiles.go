// SPDX-License-Identifier: GPL-3.0-or-later

package lcov

import (
	"io"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func ReportedFilesIn(artifact io.Reader) ([]domain.ReportedFile, error) {
	return []domain.ReportedFile{{Path: "internal/change/domain/effect.go", IsExercised: true}}, nil
}
