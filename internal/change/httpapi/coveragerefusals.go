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
)

type classifiedFailure struct {
	answer answerBody
	level  slog.Level
}

type causedRefusal struct {
	cause   error
	refusal func() refusal
}

var coverageRefusals = []causedRefusal{
	{cause: application.ErrForeignRepository, refusal: foreignRepositoryProblem},
	{cause: domain.ErrPathOutsideRepository, refusal: unknownPathProblem},
}

func coverageRefusalOf(failure error) classifiedFailure {
	matching := slices.IndexFunc(coverageRefusals, func(row causedRefusal) bool { return errors.Is(failure, row.cause) })
	if matching < 0 {
		return classifiedFailure{answer: serverProblem{Code: "coverage_failed", Message: "x"}, level: slog.LevelError}
	}
	return classifiedFailure{answer: coverageRefusals[matching].refusal(), level: slog.LevelWarn}
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
