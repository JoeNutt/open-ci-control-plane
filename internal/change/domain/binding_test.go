// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBindingKeepsWhatItStates(t *testing.T) {
	binding, err := domain.ParseBinding(thisRepositorysBinding())
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	if binding.Repository().String() != "JoeNutt/open-ci-control-plane" {
		t.Errorf("repository is %q", binding.Repository())
	}
	if binding.Workflow().String() != "open-ci-tests.yml" {
		t.Errorf("workflow is %q", binding.Workflow())
	}
	alwaysRun := binding.AlwaysRun()
	if len(alwaysRun) != 1 || alwaysRun[0].String() != thisRepositorysDomain {
		t.Errorf("always-run suites are %v, want the domain package", alwaysRun)
	}
}

func TestParseBindingKeepsEachPolicyItUses(t *testing.T) {
	binding, err := domain.ParseBinding(thisRepositorysBinding())
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}

	policies := binding.Policies()
	if len(policies) != 1 || policies[0].Use().String() != "clean-architecture" {
		t.Fatalf("policies are %v, want clean-architecture alone", policies)
	}
	roots := policies[0].ImportRoots()
	if len(roots) != 1 || roots[0].Prefix().String() != thisRepositorysModule || roots[0].Directory().String() != "" {
		t.Errorf("import roots are %v, want the module at the repository root", roots)
	}
}

func TestParseBindingKeepsEachGroupsPathsInGroupOrder(t *testing.T) {
	binding, err := domain.ParseBinding(thisRepositorysBinding())
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}

	groups := binding.Policies()[0].Groups()
	if len(groups) != 2 || groups[0].Group().String() != "adapter" || groups[1].Group().String() != "domain" {
		t.Fatalf("groups are %v, want adapter then domain", groups)
	}
	patterns := groups[1].Patterns()
	if len(patterns) != 1 || !patterns[0].Matches("internal/change/domain/effect.go") {
		t.Fatalf("domain patterns are %v, want one matching the domain package", patterns)
	}
	if patterns[0].Glob() != "internal/change/domain/**" {
		t.Errorf("domain glob is %q, want it as the binding wrote it", patterns[0].Glob())
	}
}

func TestParseBindingRefusesABindingItCannotTrust(t *testing.T) {
	refused := map[string]struct {
		stated func(*domain.BindingStatement)
		reason error
	}{
		"a malformed repository": {
			stated: func(s *domain.BindingStatement) { s.Repository = "open-ci-control-plane" },
			reason: domain.ErrMalformedRepository,
		},
		"a missing workflow": {
			stated: func(s *domain.BindingStatement) { s.Workflow = "" },
			reason: domain.ErrMissingWorkflow,
		},
		"an always-run suite that is no import path": {
			stated: func(s *domain.BindingStatement) { s.AlwaysRun = []string{"../domain"} },
			reason: domain.ErrMalformedGoImportPath,
		},
		"a malformed policy name": {
			stated: func(s *domain.BindingStatement) { s.Policies[0].Use = "clean architecture" },
			reason: domain.ErrMalformedPolicyName,
		},
		"an empty import-root prefix": {
			stated: func(s *domain.BindingStatement) { s.Policies[0].ImportRoots[0].Prefix = "" },
			reason: domain.ErrMalformedGoImportPath,
		},
		"an import root outside the repository": {
			stated: func(s *domain.BindingStatement) { s.Policies[0].ImportRoots[0].Path = "../elsewhere" },
			reason: domain.ErrMalformedRepositoryDirectory,
		},
		"two import roots sharing a prefix": {
			stated: func(s *domain.BindingStatement) {
				s.Policies[0].ImportRoots = append(s.Policies[0].ImportRoots,
					domain.ImportRootStatement{Prefix: thisRepositorysModule, Path: "internal"})
			},
			reason: domain.ErrSharedImportRootPrefix,
		},
		"a malformed group name in paths": {
			stated: func(s *domain.BindingStatement) { s.Policies[0].Paths["do main"] = []string{"cmd/**"} },
			reason: domain.ErrMalformedGroupName,
		},
		"a path pattern this project does not translate": {
			stated: func(s *domain.BindingStatement) { s.Policies[0].Paths["domain"] = []string{"internal/[ab]/**"} },
			reason: domain.ErrUnsupportedPathPattern,
		},
		"one path bound to two groups": {
			stated: func(s *domain.BindingStatement) {
				s.Policies[0].Paths["domain"] = append(s.Policies[0].Paths["domain"], "internal/change/forge/**")
			},
			reason: domain.ErrPathInTwoGroups,
		},
		"one policy used twice": {
			stated: func(s *domain.BindingStatement) { s.Policies = append(s.Policies, s.Policies[0]) },
			reason: domain.ErrDuplicateBoundPolicy,
		},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			stated := thisRepositorysBinding()
			refusal.stated(&stated)
			if _, err := domain.ParseBinding(stated); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
	}
}

const (
	thisRepositorysModule = "github.com/JoeNutt/open-ci-control-plane"
	thisRepositorysDomain = thisRepositorysModule + "/internal/change/domain"
)

func thisRepositorysBinding() domain.BindingStatement {
	return domain.BindingStatement{
		Repository: "JoeNutt/open-ci-control-plane",
		Workflow:   "open-ci-tests.yml",
		AlwaysRun:  []string{thisRepositorysDomain},
		Policies: []domain.BoundPolicyStatement{{
			Use:         "clean-architecture",
			ImportRoots: []domain.ImportRootStatement{{Prefix: thisRepositorysModule, Path: ""}},
			Paths: map[string][]string{
				"domain":  {"internal/change/domain/**"},
				"adapter": {"internal/change/forge/**", "internal/change/policy/**"},
			},
		}},
	}
}
