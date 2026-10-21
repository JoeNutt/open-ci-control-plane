// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"fmt"
)

var ErrMissingItem = errors.New("the configuration is missing a required item")

var ErrInvalidItem = errors.New("a configuration item is invalid")

type item string

func (absent item) missing() error {
	return fmt.Errorf("%w: %s", ErrMissingItem, absent)
}

func (refused item) invalid(reason error) error {
	return fmt.Errorf("%w: %s: %w", ErrInvalidItem, refused, reason)
}
