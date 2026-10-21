// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const (
	contentTypeHeader = "Content-Type"
	jsonMediaType     = "application/json"
	unwrittenAnswer   = "the answer to a delivery could not be written"
)

const (
	missingHeaderCode             = "missing_header"
	missingHeaderMessage          = "the delivery carries no such header"
	malformedHeaderCode           = "malformed_header"
	malformedHeaderMessage        = "the header exceeds its length or holds a character outside its set"
	oversizedBodyCode             = "body_too_large"
	oversizedBodyMessage          = "the delivery declares a body above the bound"
	malformedPayloadCode          = "malformed_payload"
	malformedPayloadMessage       = "the signed body is not a JSON object"
	payloadField                  = "body"
	unparseablePullRequestCode    = "unparseable_pull_request"
	unparseablePullRequestMessage = "the payload names no pull request this service accepts"
	pullRequestFields             = "repository.full_name, number"
	notAllowlistedReason          = "installation_not_allowlisted"
	failedEvaluationCode          = "evaluation_failed"
	failedEvaluationMessage       = "the change could not be evaluated"
	undeclaredLengthCode          = "undeclared_length"
	undeclaredLengthMessage       = "the delivery declares no length"
)

var deliveryLimit = fmt.Sprintf("%d bytes", ingest.MaxDeliveryBytes)

type fieldProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

func (fieldProblem) status() int { return http.StatusBadRequest }

func (problem fieldProblem) reason() string { return problem.Code }

type limitProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Limit   string `json:"limit"`
}

func (limitProblem) status() int { return http.StatusRequestEntityTooLarge }

func (problem limitProblem) reason() string { return problem.Code }

type serverProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (serverProblem) status() int { return http.StatusInternalServerError }

func missingHeaderProblem(header string) fieldProblem {
	return fieldProblem{Code: missingHeaderCode, Message: missingHeaderMessage, Field: header}
}

func malformedHeaderProblem(header string) fieldProblem {
	return fieldProblem{Code: malformedHeaderCode, Message: malformedHeaderMessage, Field: header}
}

func malformedPayloadProblem() fieldProblem {
	return fieldProblem{Code: malformedPayloadCode, Message: malformedPayloadMessage, Field: payloadField}
}

func unparseablePullRequestProblem() fieldProblem {
	return fieldProblem{
		Code:    unparseablePullRequestCode,
		Message: unparseablePullRequestMessage,
		Field:   pullRequestFields,
	}
}

func oversizedBodyProblem() limitProblem {
	return limitProblem{Code: oversizedBodyCode, Message: oversizedBodyMessage, Limit: deliveryLimit}
}

func undeclaredLengthProblem() limitProblem {
	return limitProblem{Code: undeclaredLengthCode, Message: undeclaredLengthMessage, Limit: deliveryLimit}
}

func failedEvaluationProblem() serverProblem {
	return serverProblem{Code: failedEvaluationCode, Message: failedEvaluationMessage}
}
