// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type RepositoryPaths interface {
	KnownPathsOf(ctx context.Context, repository Repository) (KnownPaths, error)
}
