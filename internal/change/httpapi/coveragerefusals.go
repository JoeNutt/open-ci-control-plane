// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "net/http"

type authorizationProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (authorizationProblem) status() int { return http.StatusForbidden }

func (problem authorizationProblem) reason() string { return problem.Code }

type classifiedFailure struct {
	answer refusal
}

func coverageRefusalOf(error) classifiedFailure {
	return classifiedFailure{answer: authorizationProblem{Code: "foreign_repository", Message: "x"}}
}
