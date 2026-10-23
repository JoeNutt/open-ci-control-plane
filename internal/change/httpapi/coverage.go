// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const coverageRoute = "PUT /v1/repositories/{owner}/{repo}/coverage/{suite...}"

type CoverageReplacement interface {
	Execute(ctx context.Context, upload application.CoverageUpload) (application.ReplacedCoverage, error)
}

type coverageUploads struct {
	log         *slog.Logger
	answering   answers
	replacement CoverageReplacement
	artifacts   CoverageArtifacts
}

func (receiving coverageUploads) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	answered := receiving.answerTo(answer, asked)
	answered.logTo(asked.Context(), receiving.log, refusedCoverage)
	receiving.answering.write(asked.Context(), answer, answered.answer)
}

func (receiving coverageUploads) answerTo(answer http.ResponseWriter, asked *http.Request) loggedAnswer {
	target, refused := coverageTargetOf(asked.PathValue(ownerSegment), asked.PathValue(repoSegment), asked.PathValue(suiteSegment))
	if refused != nil {
		return refusedBeforeUse(refused)
	}
	if refused := uploadLengthRefusalOf(asked.ContentLength); refused != nil {
		return refusedBeforeUse(refused)
	}
	asked.Body = http.MaxBytesReader(answer, asked.Body, domain.DeclaredCoverageLimits().MaxCompressedBytes)
	token, err := bearerTokenIn(asked.Header.Get(authorizationHeader))
	if err != nil {
		return coverageRefusalOf(err)
	}
	form, refused := uploadedFormOf(asked, receiving.artifacts)
	if refused != nil {
		return refusedBeforeUse(refused)
	}
	replaced, err := receiving.replacement.Execute(asked.Context(), application.CoverageUpload{
		Token: token, Repository: target.repository, Suite: target.suite, Report: form,
	})
	if err != nil {
		return coverageRefusalOf(err)
	}
	return loggedAnswer{answer: coverageAccepted{Suite: replaced.Suite.String(), FileEntryCount: replaced.FileEntryCount}}
}

type coverageAccepted struct {
	Suite          string `json:"suite"`
	FileEntryCount int    `json:"fileEntryCount"`
}

func (coverageAccepted) status() int { return http.StatusCreated }
