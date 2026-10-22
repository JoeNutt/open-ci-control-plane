// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	foreignRepositoryCode    = "foreign_repository"
	foreignRepositoryMessage = "the token was issued to another repository than the path names"
)

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

type classifiedFailure struct {
	answer refusal
}

func coverageRefusalOf(failure error) classifiedFailure {
	if errors.Is(failure, domain.ErrPathOutsideRepository) {
		return classifiedFailure{answer: pathProblem{Code: "unknown_path", Message: "x"}}
	}
	return classifiedFailure{answer: foreignRepositoryProblem()}
}
