// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

var ErrNoBinding = errors.New("the policy repository holds no binding for the repository")

type BindingSource interface {
	BindingOf(ctx context.Context, repository GovernedRepository) (Binding, error)
}

func (model PolicyModel) BindingOf(repository GovernedRepository) (Binding, error) {
	bound := slices.IndexFunc(model.bindings, func(binding Binding) bool { return binding.repository == repository })
	if bound < 0 {
		return Binding{}, fmt.Errorf("%w: %s", ErrNoBinding, repository)
	}
	return model.bindings[bound], nil
}
