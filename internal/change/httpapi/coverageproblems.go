// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	missingPartCode               = "missing_part"
	missingPartMessage            = "the form has no such part where C-12 requires it: format, then artifact"
	unknownFormatCode             = "unknown_format"
	unknownFormatMessage          = "the format is not lcov, the only format accepted"
	unparseableArtifactCode       = "unparseable_artifact"
	unparseableArtifactMessage    = "the artifact is not parseable as LCOV"
	boundExceededCode             = "bound_exceeded"
	rateExceededCode              = "rate_exceeded"
	rateExceededMessage           = "the repository has uploaded more coverage this minute than C-12 allows"
	retryAfterHeader              = "Retry-After"
	boundExceededMessage          = "the artifact exceeds an NFR-5 or NFR-13 bound; nothing was stored"
)

var declaredCoverageLimits = domain.DeclaredCoverageLimits()

var (
	compressedArtifactLimit   = fmt.Sprintf("%d compressed bytes", declaredCoverageLimits.MaxCompressedBytes)
	decompressedArtifactLimit = fmt.Sprintf("%d decompressed bytes", declaredCoverageLimits.MaxDecompressedBytes)
	fileEntriesLimit          = fmt.Sprintf("%d file entries", declaredCoverageLimits.MaxFileEntries)
	suitesLimit               = fmt.Sprintf("%d suites per repository", declaredCoverageLimits.MaxSuitesPerRepository)
	testSectionsLimit         = fmt.Sprintf("%d TN sections", declaredCoverageLimits.MaxTestSections)
	lineRecordsLimit          = fmt.Sprintf("%d DA records", declaredCoverageLimits.MaxLineRecords)
)

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

func missingPartProblem(part string) refusal {
	return fieldProblem{Code: missingPartCode, Message: missingPartMessage, Field: part}
}

func unknownFormatProblem() refusal {
	return fieldProblem{Code: unknownFormatCode, Message: unknownFormatMessage, Field: formatPart}
}

func unparseableArtifactProblem() refusal {
	return fieldProblem{Code: unparseableArtifactCode, Message: unparseableArtifactMessage, Field: artifactPart}
}

func boundProblem(limit string) refusal {
	return limitProblem{Code: boundExceededCode, Message: boundExceededMessage, Limit: limit}
}

type rateProblem struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	retryAfter time.Duration
}

func (rateProblem) status() int { return http.StatusTooManyRequests }

func (problem rateProblem) reason() string { return problem.Code }

func (problem rateProblem) writeHeaders(headers http.Header) {
	headers.Set(retryAfterHeader, strconv.FormatInt(int64(problem.retryAfter/time.Second), 10))
}

func rateProblemAfter(retryAfter time.Duration) refusal {
	return rateProblem{Code: rateExceededCode, Message: rateExceededMessage, retryAfter: retryAfter}
}
