// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"slices"
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

func TestParsePolicyListingHoldsAFileListedAtTheByteBoundAndRefusesOneByteOver(t *testing.T) {
	atLimit := listedPolicyFiles(1)
	atLimit[0].Bytes = domain.MaxPolicyFileBytes
	if _, err := domain.ParsePolicyListing(atLimit); err != nil {
		t.Fatalf("a file listed at %d bytes, the limit, was refused: %v", domain.MaxPolicyFileBytes, err)
	}

	overLimit := listedPolicyFiles(1)
	overLimit[0].Bytes = domain.MaxPolicyFileBytes + 1
	if _, err := domain.ParsePolicyListing(overLimit); !errors.Is(err, domain.ErrPolicyFileTooLarge) {
		t.Fatalf("a file listed at %d bytes gave %v, want ErrPolicyFileTooLarge", domain.MaxPolicyFileBytes+1, err)
	}
}

func TestParsePolicyListingHoldsEveryListedFileInOrder(t *testing.T) {
	listed := listedPolicyFiles(3)

	listing, err := domain.ParsePolicyListing(listed)

	if err != nil {
		t.Fatalf("parse policy listing: %v", err)
	}
	if !slices.Equal(listing.Files(), listed) {
		t.Errorf("the listing holds %+v, want %+v", listing.Files(), listed)
	}
}

func TestParsePolicyListingHandsOutEachListedPathParsed(t *testing.T) {
	listed := listedPolicyFiles(2)

	listing, err := domain.ParsePolicyListing(listed)
	if err != nil {
		t.Fatalf("parse policy listing: %v", err)
	}

	paths := listing.Paths()
	if len(paths) != len(listed) || paths[0].String() != listed[0].Path || paths[1].String() != listed[1].Path {
		t.Errorf("the listing's paths are %v, want those of %+v", paths, listed)
	}
}

func TestParsePolicyListingRefusesAFileListedWithANegativeSize(t *testing.T) {
	listed := listedPolicyFiles(1)
	listed[0].Bytes = -1

	if _, err := domain.ParsePolicyListing(listed); !errors.Is(err, domain.ErrMalformedPolicyFileSize) {
		t.Fatalf("a file listed at -1 bytes gave %v, want ErrMalformedPolicyFileSize", err)
	}
}

func TestParsePolicyListingRefusesAPathOutsideTheRepository(t *testing.T) {
	for _, path := range []string{"", "/catalog/x.yaml", "catalog/../../x.yaml", "catalog//x.yaml"} {
		listed := listedPolicyFiles(1)
		listed[0].Path = path
		if _, err := domain.ParsePolicyListing(listed); !errors.Is(err, domain.ErrMalformedPolicyFilePath) {
			t.Errorf("listed path %q gave %v, want ErrMalformedPolicyFilePath", path, err)
		}
	}
}

func listedPolicyFiles(count int) []domain.ListedFile {
	listed := make([]domain.ListedFile, 0, count)
	for index := range count {
		listed = append(listed, domain.ListedFile{Path: fmt.Sprintf(boundPolicyFilePathFormat, index), Bytes: 1})
	}
	return listed
}
