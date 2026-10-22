// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBoundedPolicyFilesHoldsTheFileCountAtItsLimitAndRefusesOneOver(t *testing.T) {
	if _, err := domain.ParseBoundedPolicyFiles(policyFiles(domain.MaxPolicyFiles)); err != nil {
		t.Fatalf("%d files, the limit, were refused: %v", domain.MaxPolicyFiles, err)
	}
	_, err := domain.ParseBoundedPolicyFiles(policyFiles(domain.MaxPolicyFiles + 1))
	if !errors.Is(err, domain.ErrTooManyPolicyFiles) {
		t.Fatalf("%d files gave %v, want ErrTooManyPolicyFiles", domain.MaxPolicyFiles+1, err)
	}
}

func TestParseBoundedPolicyFilesHoldsAFileAtTheByteBoundAndRefusesOneByteOver(t *testing.T) {
	atLimit := policyFiles(1)
	atLimit[0].Content = make([]byte, domain.MaxPolicyFileBytes)
	if _, err := domain.ParseBoundedPolicyFiles(atLimit); err != nil {
		t.Fatalf("a %d-byte file, the limit, was refused: %v", domain.MaxPolicyFileBytes, err)
	}

	overLimit := policyFiles(1)
	overLimit[0].Content = make([]byte, domain.MaxPolicyFileBytes+1)
	if _, err := domain.ParseBoundedPolicyFiles(overLimit); !errors.Is(err, domain.ErrPolicyFileTooLarge) {
		t.Fatalf("a %d-byte file gave %v, want ErrPolicyFileTooLarge", domain.MaxPolicyFileBytes+1, err)
	}
}

func TestParseBoundedPolicyFilesRefusesAPathOutsideTheRepository(t *testing.T) {
	for _, path := range []string{"", "/catalog/x.yaml", "catalog/../../x.yaml", "catalog//x.yaml"} {
		files := policyFiles(1)
		files[0].Path = path
		if _, err := domain.ParseBoundedPolicyFiles(files); !errors.Is(err, domain.ErrMalformedPolicyFilePath) {
			t.Errorf("path %q gave %v, want ErrMalformedPolicyFilePath", path, err)
		}
	}
}

const boundPolicyFilePathFormat = "bindings/owner/repository%d.yaml"

func policyFiles(count int) []domain.PolicyFile {
	files := make([]domain.PolicyFile, 0, count)
	for index := range count {
		files = append(files, domain.PolicyFile{
			Path:    fmt.Sprintf(boundPolicyFilePathFormat, index),
			Content: []byte("schema_version: 1\n"),
		})
	}
	return files
}
