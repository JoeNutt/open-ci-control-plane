// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBindingHoldsEachBoundAtItsLimitAndRefusesOneOver(t *testing.T) {
	bounds := map[string]struct {
		limit  int
		fill   func(*domain.BindingStatement, int)
		breach error
	}{
		"module groups": {
			limit: domain.MaxBindingGroups,
			fill: func(s *domain.BindingStatement, count int) {
				s.Policies[0].Paths = map[string][]string{}
				for index := range count {
					s.Policies[0].Paths[fmt.Sprintf("group%d", index)] = []string{fmt.Sprintf("g%d/**", index)}
				}
			},
			breach: domain.ErrTooManyBindingGroups,
		},
		"path patterns": {
			limit: domain.MaxBindingPathPatterns,
			fill: func(s *domain.BindingStatement, count int) {
				s.Policies[0].Paths = map[string][]string{"domain": distinct("p%d/**", count)}
			},
			breach: domain.ErrTooManyBindingPathPatterns,
		},
		"import roots": {
			limit: domain.MaxBindingImportRoots,
			fill: func(s *domain.BindingStatement, count int) {
				s.Policies[0].ImportRoots = nil
				for _, prefix := range distinct("example.com/root%d", count) {
					s.Policies[0].ImportRoots = append(s.Policies[0].ImportRoots,
						domain.ImportRootStatement{Prefix: prefix})
				}
			},
			breach: domain.ErrTooManyBindingImportRoots,
		},
		"always-run suites": {
			limit: domain.MaxBindingAlwaysRunSuites,
			fill: func(s *domain.BindingStatement, count int) {
				s.AlwaysRun = distinct("example.com/suite%d", count)
			},
			breach: domain.ErrTooManyAlwaysRunSuites,
		},
	}
	for bound, rule := range bounds {
		t.Run(bound, func(t *testing.T) {
			atLimit := thisRepositorysBinding()
			rule.fill(&atLimit, rule.limit)
			if _, err := domain.ParseBinding(atLimit); err != nil {
				t.Fatalf("%d %s, the limit, was refused: %v", rule.limit, bound, err)
			}

			overLimit := thisRepositorysBinding()
			rule.fill(&overLimit, rule.limit+1)
			if _, err := domain.ParseBinding(overLimit); !errors.Is(err, rule.breach) {
				t.Fatalf("%d %s gave %v, want %v", rule.limit+1, bound, err, rule.breach)
			}
		})
	}
}

func distinct(format string, count int) []string {
	written := make([]string, 0, count)
	for index := range count {
		written = append(written, fmt.Sprintf(format, index))
	}
	return written
}
