// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

const firstPullRequestNumber = 1

var ErrMalformedRepository = errors.New("repository is not an owner/name pair")

var ErrMalformedPullRequestNumber = errors.New("pull request number is not positive")

var ErrNoSuchPullRequest = errors.New("the forge reports no such pull request")

type PullRequestLocator struct {
	repository GovernedRepository
	number     int
}

func ParsePullRequestLocator(repository string, number int) (PullRequestLocator, error) {
	parsed, err := ParseGovernedRepository(repository)
	if err != nil {
		return PullRequestLocator{}, fmt.Errorf("pull request locator: %w", err)
	}
	if number < firstPullRequestNumber {
		return PullRequestLocator{}, fmt.Errorf("%w: %d", ErrMalformedPullRequestNumber, number)
	}
	return PullRequestLocator{repository: parsed, number: number}, nil
}

func (locator PullRequestLocator) String() string {
	return fmt.Sprintf("%s#%d", locator.Repository(), locator.number)
}

func (locator PullRequestLocator) Repository() string { return locator.repository.String() }

func (locator PullRequestLocator) GovernedRepository() GovernedRepository { return locator.repository }

func (locator PullRequestLocator) Owner() string { return locator.repository.Owner() }

func (locator PullRequestLocator) Name() string { return locator.repository.Name() }

func (locator PullRequestLocator) Number() int { return locator.number }
