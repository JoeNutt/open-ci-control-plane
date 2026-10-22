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
	formatPart         = "format"
	artifactPart       = "artifact"
	lcovFormat         = "lcov"
	maxFormatReadBytes = int64(len(lcovFormat)) + 1
)

type CoverageArtifacts interface {
	ArtifactOf(declaredBytes int64, body io.Reader) domain.CoverageReport
}

var errUnknownFormat = errors.New("the form states a format other than lcov")

var (
	errNoFormatPart   = errors.New("the form has no format part where one is due")
	errNoArtifactPart = errors.New("the form has no artifact part where one is due")
)

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
	artifact, err := form.lcovArtifact()
	if err != nil {
		return nil, err
	}
	return form.artifacts.ArtifactOf(form.declared, artifact).ReportedFiles(limits)
}

func (form uploadedForm) lcovArtifact() (*multipart.Part, error) {
	format, err := form.nextPartNamed(formatPart, errNoFormatPart)
	if err != nil {
		return nil, err
	}
	if err := checkLCOVFormat(format); err != nil {
		return nil, err
	}
	return form.nextPartNamed(artifactPart, errNoArtifactPart)
}

func checkLCOVFormat(format io.Reader) error {
	stated, err := io.ReadAll(io.LimitReader(format, maxFormatReadBytes))
	if err != nil || string(stated) != lcovFormat {
		return errUnknownFormat
	}
	return nil
}

func (form uploadedForm) nextPartNamed(name string, missing error) (*multipart.Part, error) {
	next, err := form.parts.NextPart()
	if err != nil || next.FormName() != name {
		return nil, missing
	}
	return next, nil
}
