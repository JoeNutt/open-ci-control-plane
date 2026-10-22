// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
)

var ErrNoBinding = errors.New("the policy repository holds no binding for the repository")

type BindingSource interface {
	BindingOf(ctx context.Context, repository GovernedRepository) (Binding, error)
}
