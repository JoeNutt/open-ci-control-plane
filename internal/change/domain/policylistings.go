// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type PolicyListings interface {
	ListedFilesAt(ctx context.Context, at PolicyCommit) ([]ListedFile, error)
}
