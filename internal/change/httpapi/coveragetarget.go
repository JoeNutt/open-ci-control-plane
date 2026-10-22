// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

const (
	ownerSegment       = "owner"
	repoSegment        = "repo"
	suiteSegment       = "suite"
	repositorySegments = ownerSegment + "/" + repoSegment
)

type coverageTarget struct {
	repository domain.GovernedRepository
	suite      domain.CoverageSuite
}

func coverageTargetOf(owner, repo, suite string) (coverageTarget, refusal) {
	repository, err := domain.ParseGovernedRepositorySegments(owner, repo)
	if err != nil {
		return coverageTarget{}, malformedPathProblem(repositorySegments)
	}
	parsedSuite, err := domain.ParseCoverageSuite(suite)
	if err != nil {
		return coverageTarget{}, malformedPathProblem(suiteSegment)
	}
	return coverageTarget{repository: repository, suite: parsedSuite}, nil
}
