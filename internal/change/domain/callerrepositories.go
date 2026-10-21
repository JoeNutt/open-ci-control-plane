// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type CallerRepositories interface {
	RepositoryOf(ctx context.Context, token IdentityToken) (GovernedRepository, error)
}
