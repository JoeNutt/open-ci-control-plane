// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	malformedPathCode             = "malformed_path"
	malformedPathMessage          = "the path segment holds a character outside its set, exceeds its length or is not one canonical segment"
	undeclaredUploadLengthMessage = "the upload declares no length"
	oversizedUploadCode           = "upload_too_large"
	oversizedUploadMessage        = "the upload declares a length above the compressed artifact bound"
	notAFormCode                  = "not_a_form"
	notAFormMessage               = "the upload is not multipart/form-data"
	foreignRepositoryCode         = "foreign_repository"
	foreignRepositoryMessage      = "the token was issued to another repository than the path names"
	unknownPathCode               = "unknown_path"
	unknownPathMessage            = "the artifact names a path that is absolute or that the repository does not hold"
	unverifiedTokenCode           = "unverified_token"
	unverifiedTokenMessage        = "the upload carries no token, or one that is expired or cannot be verified"
	failedCoverageCode            = "coverage_failed"
	failedCoverageMessage         = "the coverage could not be replaced"
)

var compressedArtifactLimit = fmt.Sprintf("%d compressed bytes", domain.DeclaredCoverageLimits().MaxCompressedBytes)

func malformedPathProblem(segment string) refusal {
	return fieldProblem{Code: malformedPathCode, Message: malformedPathMessage, Field: segment}
}

func notAFormProblem() refusal {
	return fieldProblem{Code: notAFormCode, Message: notAFormMessage, Field: contentTypeHeader}
}

func undeclaredUploadLengthProblem() refusal {
	return limitProblem{Code: undeclaredLengthCode, Message: undeclaredUploadLengthMessage, Limit: compressedArtifactLimit}
}

func oversizedUploadProblem() refusal {
	return limitProblem{Code: oversizedUploadCode, Message: oversizedUploadMessage, Limit: compressedArtifactLimit}
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
