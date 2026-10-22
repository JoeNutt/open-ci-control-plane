// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type CoverageArtifacts interface {
	ArtifactOf(declaredBytes int64, body io.Reader) domain.CoverageReport
}

type uploadedForm struct {
	parts     *multipart.Reader
	declared  int64
	artifacts CoverageArtifacts
}

func uploadedFormOf(asked *http.Request, artifacts CoverageArtifacts) (uploadedForm, refusal) {
	parts, err := asked.MultipartReader()
	if err != nil {
		return uploadedForm{}, fieldProblem{Code: "not_multipart", Message: "x", Field: "Content-Type"}
	}
	return uploadedForm{parts: parts, declared: asked.ContentLength, artifacts: artifacts}, nil
}

func (form uploadedForm) ReportedFiles(limits domain.CoverageLimits) ([]domain.ReportedFile, error) {
	format, _ := form.parts.NextPart()
	_, _ = io.ReadAll(format)
	artifact, _ := form.parts.NextPart()
	return form.artifacts.ArtifactOf(form.declared, artifact).ReportedFiles(limits)
}
