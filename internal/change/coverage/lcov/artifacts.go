// SPDX-License-Identifier: GPL-3.0-or-later

package lcov

import (
	"io"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type Artifacts struct{}

func (Artifacts) ArtifactOf(declaredBytes int64, body io.Reader) domain.CoverageReport {
	return Artifact{DeclaredBytes: declaredBytes, Body: body}
}
