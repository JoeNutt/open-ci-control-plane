// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const uploadedArtifact = "SF:internal/change/domain/effect.go\nDA:1,1\nend_of_record\n"

func TestAnUploadedFormHandsItsArtifactAndDeclaredLengthToTheArtifactParser(t *testing.T) {
	upload := formUpload(t, formPart{"format", "lcov"}, formPart{"artifact", uploadedArtifact})
	parser := &recordingArtifacts{}

	form, refused := uploadedFormOf(upload, parser)
	if refused != nil {
		t.Fatalf("refused %#v", refused)
	}
	if _, err := form.ReportedFiles(domain.DeclaredCoverageLimits()); err != nil {
		t.Fatalf("report: %v", err)
	}

	if parser.artifact != uploadedArtifact || parser.declaredBytes != upload.ContentLength {
		t.Errorf("parser was given %q declaring %d, want the artifact declaring %d",
			parser.artifact, parser.declaredBytes, upload.ContentLength)
	}
}

func TestAnUploadThatIsNotAMultipartFormIsRefused400NamingItsContentType(t *testing.T) {
	upload := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(uploadedArtifact))
	upload.Header.Set("Content-Type", "text/plain")

	_, refused := uploadedFormOf(upload, &recordingArtifacts{})

	if refused == nil || refused.status() != http.StatusBadRequest || refused.(fieldProblem).Field != "Content-Type" {
		t.Errorf("refused %#v, want a 400 naming Content-Type", refused)
	}
}

type formPart struct{ name, content string }

func formUpload(t *testing.T, parts ...formPart) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, part := range parts {
		writer, err := form.CreateFormFile(part.name, part.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(part.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	upload := httptest.NewRequest(http.MethodPut, "/", &body)
	upload.Header.Set("Content-Type", form.FormDataContentType())
	return upload
}

type recordingArtifacts struct {
	declaredBytes int64
	artifact      string
}

func (recording *recordingArtifacts) ArtifactOf(declaredBytes int64, body io.Reader) domain.CoverageReport {
	recording.declaredBytes = declaredBytes
	return recordedReport{recording: recording, body: body}
}

type recordedReport struct {
	recording *recordingArtifacts
	body      io.Reader
}

func (report recordedReport) ReportedFiles(domain.CoverageLimits) ([]domain.ReportedFile, error) {
	content, err := io.ReadAll(report.body)
	report.recording.artifact = string(content)
	return nil, err
}
