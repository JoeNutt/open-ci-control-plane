// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	foreignRepositoryCode    = "foreign_repository"
	foreignRepositoryMessage = "the token was issued to another repository than the path names"
	unknownPathCode          = "unknown_path"
	unknownPathMessage       = "the artifact names a path that is absolute or that the repository does not hold"
	unverifiedTokenCode      = "unverified_token"
	unverifiedTokenMessage   = "the upload carries no token, or one that is expired or cannot be verified"
	failedCoverageCode       = "coverage_failed"
	failedCoverageMessage    = "the coverage could not be replaced"
)

type classifiedFailure struct {
	answer answerBody
	level  slog.Level
}

type causedRefusal struct {
	cause   error
	refusal func() refusal
	level   slog.Level
}

var coverageRefusals = []causedRefusal{
	{cause: domain.ErrUnverifiedIdentityToken, refusal: unverifiedTokenProblem, level: slog.LevelWarn},
	{cause: domain.ErrIdentityIssuerUnavailable, refusal: unverifiedTokenProblem, level: slog.LevelError},
	{cause: application.ErrForeignRepository, refusal: foreignRepositoryProblem, level: slog.LevelWarn},
	{cause: domain.ErrPathOutsideRepository, refusal: unknownPathProblem, level: slog.LevelWarn},
}

func coverageRefusalOf(failure error) classifiedFailure {
	matching := slices.IndexFunc(coverageRefusals, func(row causedRefusal) bool { return errors.Is(failure, row.cause) })
	if matching < 0 {
		return classifiedFailure{answer: failedCoverageProblem(), level: slog.LevelError}
	}
	row := coverageRefusals[matching]
	return classifiedFailure{answer: row.refusal(), level: row.level}
}

type authorizationProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (authorizationProblem) status() int { return http.StatusForbidden }

func (problem authorizationProblem) reason() string { return problem.Code }

func foreignRepositoryProblem() refusal {
	return authorizationProblem{Code: foreignRepositoryCode, Message: foreignRepositoryMessage}
}

type pathProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (pathProblem) status() int { return http.StatusUnprocessableEntity }

func (problem pathProblem) reason() string { return problem.Code }

func unknownPathProblem() refusal {
	return pathProblem{Code: unknownPathCode, Message: unknownPathMessage}
}

func failedCoverageProblem() serverProblem {
	return serverProblem{Code: failedCoverageCode, Message: failedCoverageMessage}
}

type authenticationProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (authenticationProblem) status() int { return http.StatusUnauthorized }

func (problem authenticationProblem) reason() string { return problem.Code }

func unverifiedTokenProblem() refusal {
	return authenticationProblem{Code: unverifiedTokenCode, Message: unverifiedTokenMessage}
}
