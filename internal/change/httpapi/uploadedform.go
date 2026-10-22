// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	formatPart   = "format"
	artifactPart = "artifact"
)

type CoverageArtifacts interface {
	ArtifactOf(declaredBytes int64, body io.Reader) domain.CoverageReport
}

type missingPartError struct{ part string }

func (missing missingPartError) Error() string {
	return "the form has no " + missing.part + " part where one is due"
}

type uploadedForm struct {
	parts     *multipart.Reader
	declared  int64
	artifacts CoverageArtifacts
}

func uploadedFormOf(asked *http.Request, artifacts CoverageArtifacts) (uploadedForm, refusal) {
	parts, err := asked.MultipartReader()
	if err != nil {
		return uploadedForm{}, notAFormProblem()
	}
	return uploadedForm{parts: parts, declared: asked.ContentLength, artifacts: artifacts}, nil
}

func (form uploadedForm) ReportedFiles(limits domain.CoverageLimits) ([]domain.ReportedFile, error) {
	format, err := form.nextPartNamed(formatPart)
	if err != nil {
		return nil, err
	}
	_, _ = io.ReadAll(format)
	artifact, err := form.nextPartNamed(artifactPart)
	if err != nil {
		return nil, err
	}
	return form.artifacts.ArtifactOf(form.declared, artifact).ReportedFiles(limits)
}

func (form uploadedForm) nextPartNamed(name string) (*multipart.Part, error) {
	next, err := form.parts.NextPart()
	if err != nil || next.FormName() != name {
		return nil, missingPartError{part: name}
	}
	return next, nil
}
