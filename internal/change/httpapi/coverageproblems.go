// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"

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
