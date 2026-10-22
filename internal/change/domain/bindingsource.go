// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type BindingSource interface {
	BindingOf(ctx context.Context, repository GovernedRepository) (Binding, error)
}
