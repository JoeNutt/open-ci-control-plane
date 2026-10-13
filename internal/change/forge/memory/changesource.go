// SPDX-License-Identifier: GPL-3.0-or-later

// Package memory answers for a forge from values a test recorded.
package memory

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeSource struct {
	Files map[domain.PullRequestLocator][]domain.File
}

func (source ChangeSource) FilesChanged(
	_ context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	files, recorded := source.Files[locator]
	if !recorded {
		return nil, fmt.Errorf("%w: %s/%s#%d", domain.ErrNoSuchPullRequest,
			locator.Owner(), locator.Name(), locator.Number())
	}
	return files, nil
}
