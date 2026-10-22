// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const fileCommit = "4b9c4930c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6"

func governedRepositoryOf(t *testing.T, stated string) domain.GovernedRepository {
	t.Helper()

	repository, err := domain.ParseGovernedRepository(stated)
	if err != nil {
		t.Fatalf("parse repository %q: %v", stated, err)
	}
	return repository
}

func mustParseCommitSHA(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %q: %v", stated, err)
	}
	return commit
}

func TestParseFileAtCommitHoldsTheParsedRepositoryCommitAndPath(t *testing.T) {
	repository := governedRepositoryOf(t, "acme/payments")
	commit := mustParseCommitSHA(t, fileCommit)

	file, err := domain.ParseFileAtCommit(repository, commit, "internal/auth/middleware.go")

	if err != nil {
		t.Fatalf("parse file at commit: %v", err)
	}
	if file.Repository() != repository || file.Commit() != commit ||
		file.Path().String() != "internal/auth/middleware.go" {
		t.Errorf("file is %s, want internal/auth/middleware.go in acme/payments at %s", file, fileCommit)
	}
}

func TestParseFileAtCommitRefusesAPartThatIsMissingOrNotInTheRepository(t *testing.T) {
	repository := governedRepositoryOf(t, "acme/payments")
	commit := mustParseCommitSHA(t, fileCommit)
	refused := map[string]struct {
		repository domain.GovernedRepository
		commit     domain.CommitSHA
		path       string
		want       error
	}{
		"no repository":        {domain.GovernedRepository{}, commit, "main.go", domain.ErrMalformedRepository},
		"no commit":            {repository, domain.CommitSHA{}, "main.go", domain.ErrMalformedCommitSHA},
		"climbing out":         {repository, commit, "../secrets.go", domain.ErrPathOutsideRepository},
		"absolute":             {repository, commit, "/etc/passwd", domain.ErrPathOutsideRepository},
		"query syntax":         {repository, commit, "internal/a?ref=main.go", domain.ErrPathOutsideRepository},
		"fragment syntax":      {repository, commit, "internal/a#b.go", domain.ErrPathOutsideRepository},
		"escape syntax":        {repository, commit, "internal/b%2e.go", domain.ErrPathOutsideRepository},
		"an empty segment":     {repository, commit, "internal//main.go", domain.ErrPathOutsideRepository},
		"a trailing separator": {repository, commit, "internal/", domain.ErrPathOutsideRepository},
	}

	for name, stated := range refused {
		_, err := domain.ParseFileAtCommit(stated.repository, stated.commit, stated.path)

		if !errors.Is(err, stated.want) {
			t.Errorf("%s: error is %v, want %v", name, err, stated.want)
		}
	}
}
