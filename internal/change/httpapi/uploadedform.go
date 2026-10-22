// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
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

var errUnknownFormat = errors.New("unknown format")

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
	stated, err := io.ReadAll(io.LimitReader(format, 5))
	if err != nil || string(stated) != "lcov" {
		return nil, errUnknownFormat
	}
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
