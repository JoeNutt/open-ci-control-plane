// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestEachDeclaredChangeBoundAdmitsItsLimitAndRefusesOneOver(t *testing.T) {
	limits := domain.DeclaredLimits()
	bounds := map[string]struct {
		changeOf func(measure int) []domain.File
		limit    int
		breach   error
	}{
		"changed files": {
			changeOf: func(count int) []domain.File { return make([]domain.File, count) },
			limit:    limits.MaxChangedFiles, breach: domain.ErrTooManyChangedFiles,
		},
		"patch bytes per file": {
			changeOf: func(size int) []domain.File { return []domain.File{{Path: "a.go", PatchBytes: size}} },
			limit:    limits.MaxPatchBytesPerFile, breach: domain.ErrFilePatchTooLarge,
		},
		"path bytes": {
			changeOf: func(size int) []domain.File { return []domain.File{{Path: strings.Repeat("a", size)}} },
			limit:    limits.MaxPathBytes, breach: domain.ErrPathTooLong,
		},
		"path segments": {
			changeOf: func(depth int) []domain.File {
				return []domain.File{{Path: strings.Repeat("a/", depth-1) + "a.go"}}
			},
			limit: limits.MaxPathSegments, breach: domain.ErrPathTooDeep,
		},
	}

	for name, bound := range bounds {
		t.Run(name, func(t *testing.T) {
			if _, err := domain.ParseBoundedFiles(bound.changeOf(bound.limit), limits); err != nil {
				t.Errorf("at the limit of %d: %v", bound.limit, err)
			}
			if _, err := domain.ParseBoundedFiles(bound.changeOf(bound.limit+1), limits); !errors.Is(err, bound.breach) {
				t.Errorf("one over the limit of %d: error is %v, want %v", bound.limit, err, bound.breach)
			}
		})
	}
}
