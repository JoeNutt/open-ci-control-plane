// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "net/http"

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

type classifiedFailure struct {
	answer refusal
}

func coverageRefusalOf(error) classifiedFailure {
	return classifiedFailure{answer: foreignRepositoryProblem()}
}
