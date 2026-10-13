// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "strings"

type PullRequestLocator struct {
	owner  string
	name   string
	number int
}

func ParsePullRequestLocator(repository string, number int) (PullRequestLocator, error) {
	owner, name, _ := strings.Cut(repository, "/")
	return PullRequestLocator{owner: owner, name: name, number: number}, nil
}

func (locator PullRequestLocator) Owner() string { return locator.owner }

func (locator PullRequestLocator) Name() string { return locator.name }

func (locator PullRequestLocator) Number() int { return locator.number }
