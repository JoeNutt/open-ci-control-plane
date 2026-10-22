// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type PolicyContents interface {
	ContentsAt(ctx context.Context, at PolicyCommit, path RepositoryPath) ([]byte, error)
}
