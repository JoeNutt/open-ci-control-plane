// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"fmt"
)

var ErrMissingItem = errors.New("the configuration is missing a required item")

var ErrInvalidItem = errors.New("a configuration item is invalid")

var errEmptyList = errors.New("the list names nothing")

var errNonPositiveID = errors.New("a GitHub ID is not positive")

const firstGitHubID = 1

type item string

func (named item) missing() error {
	return fmt.Errorf("%w: %s", ErrMissingItem, named)
}

func (named item) invalid(reason error) error {
	return fmt.Errorf("%w: %s: %w", ErrInvalidItem, named, reason)
}
