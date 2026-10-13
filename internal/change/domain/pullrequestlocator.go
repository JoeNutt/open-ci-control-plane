// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	repositorySeparator  = "/"
	maxRepositorySegment = 100
	maxRepositoryLength  = 2*maxRepositorySegment + len(repositorySeparator)
)

var repositorySegmentPattern = fmt.Sprintf(
	`[A-Za-z0-9][A-Za-z0-9._-]{0,%d}`, maxRepositorySegment-1)

var repositoryPattern = regexp.MustCompile(
	"^" + repositorySegmentPattern + repositorySeparator + repositorySegmentPattern + "$")

var ErrMalformedRepository = errors.New("repository is not an owner/name pair")

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
	if !repositoryPattern.MatchString(repository) {
		return PullRequestLocator{}, fmt.Errorf("%w: %q", ErrMalformedRepository, repository)
	}
	owner, name, _ := strings.Cut(repository, repositorySeparator)
	return PullRequestLocator{owner: owner, name: name, number: number}, nil
}

func (locator PullRequestLocator) Owner() string { return locator.owner }

func (locator PullRequestLocator) Name() string { return locator.name }

func (locator PullRequestLocator) Number() int { return locator.number }
