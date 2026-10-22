// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParsePolicyListingHoldsTheFileCountAtItsLimitAndRefusesOneOver(t *testing.T) {
	if _, err := domain.ParsePolicyListing(listedPolicyFiles(domain.MaxPolicyFiles)); err != nil {
		t.Fatalf("%d listed files, the limit, were refused: %v", domain.MaxPolicyFiles, err)
	}
	_, err := domain.ParsePolicyListing(listedPolicyFiles(domain.MaxPolicyFiles + 1))
	if !errors.Is(err, domain.ErrTooManyPolicyFiles) {
		t.Fatalf("%d listed files gave %v, want ErrTooManyPolicyFiles", domain.MaxPolicyFiles+1, err)
	}
}

func listedPolicyFiles(count int) []domain.ListedFile {
	listed := make([]domain.ListedFile, 0, count)
	for _, file := range policyFiles(count) {
		listed = append(listed, domain.ListedFile{Path: file.Path, Bytes: len(file.Content)})
	}
	return listed
}
