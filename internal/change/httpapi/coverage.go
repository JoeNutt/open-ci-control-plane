// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
)

const coverageRoute = "PUT /v1/repositories/{owner}/{repo}/coverage/{suite...}"

type CoverageReplacement interface {
	Execute(ctx context.Context, upload application.CoverageUpload) (application.ReplacedCoverage, error)
}

type coverageUploads struct {
	answering   answers
	replacement CoverageReplacement
	artifacts   CoverageArtifacts
}

func (receiving coverageUploads) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	receiving.answering.write(asked.Context(), answer, receiving.answerTo(asked))
}

func (receiving coverageUploads) answerTo(asked *http.Request) answerBody {
	target, refused := coverageTargetOf(asked.PathValue(ownerSegment), asked.PathValue(repoSegment), asked.PathValue(suiteSegment))
	if refused != nil {
		return refused
	}
	token, err := bearerTokenIn(asked.Header.Get(authorizationHeader))
	if err != nil {
		return coverageRefusalOf(err).answer
	}
	form, refused := uploadedFormOf(asked, receiving.artifacts)
	if refused != nil {
		return refused
	}
	replaced, err := receiving.replacement.Execute(asked.Context(), application.CoverageUpload{
		Token: token, Repository: target.repository, Suite: target.suite, Report: form,
	})
	if err != nil {
		return coverageRefusalOf(err).answer
	}
	return coverageAccepted{Suite: replaced.Suite.String(), FileEntryCount: replaced.FileEntryCount}
}

type coverageAccepted struct {
	Suite          string `json:"suite"`
	FileEntryCount int    `json:"fileEntryCount"`
}

func (coverageAccepted) status() int { return http.StatusCreated }
