// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const firstPullRequestNumber = 1

var ErrMalformedRepository = errors.New("repository is not an owner/name pair")

var ErrMalformedPullRequestNumber = errors.New("pull request number is not positive")

var ErrNoSuchPullRequest = errors.New("the forge reports no such pull request")

type PullRequestLocator struct {
	owner  string
	name   string
	number int
}

func ParsePullRequestLocator(repository string, number int) (PullRequestLocator, error) {
	if len(repository) > maxRepositoryLength {
		return PullRequestLocator{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrMalformedRepository, len(repository), maxRepositoryLength)
	}
	owner, name, separated := strings.Cut(repository, repositorySeparator)
	if !separated || !repositorySegmentOK(owner) || !nameSegmentOK(name) {
		return PullRequestLocator{}, fmt.Errorf("%w: %q", ErrMalformedRepository, repository)
	}
	if number < firstPullRequestNumber {
		return PullRequestLocator{}, fmt.Errorf("%w: %d", ErrMalformedPullRequestNumber, number)
	}
	return PullRequestLocator{owner: owner, name: name, number: number}, nil
}

func (locator PullRequestLocator) String() string {
	return fmt.Sprintf("%s#%d", locator.Repository(), locator.number)
}

func (locator PullRequestLocator) Repository() string {
	return locator.owner + repositorySeparator + locator.name
}

func (locator PullRequestLocator) Owner() string { return locator.owner }

func (locator PullRequestLocator) Name() string { return locator.name }

func (locator PullRequestLocator) Number() int { return locator.number }
