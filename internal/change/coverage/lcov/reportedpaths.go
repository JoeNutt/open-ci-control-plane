// SPDX-License-Identifier: GPL-3.0-or-later

package lcov

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

type reportedPaths map[domain.RepositoryPath]bool

func (reported reportedPaths) entriesWith(path domain.RepositoryPath) int {
	if reported[path] {
		return len(reported)
	}
	return len(reported) + 1
}
