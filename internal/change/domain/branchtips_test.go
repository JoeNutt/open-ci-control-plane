// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	baseTipCommit = "7c6b5a4938271605f4e3d2c1b0a9f8e7d6c5b4a3"
	headTipCommit = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"
)

func TestParseBranchTipsHoldsTheBaseAndHeadItWasGiven(t *testing.T) {
	base, head := mustParseCommitSHA(t, baseTipCommit), mustParseCommitSHA(t, headTipCommit)

	tips, err := domain.ParseBranchTips(base, head)

	if err != nil {
		t.Fatalf("parse branch tips: %v", err)
	}
	if tips.Base() != base || tips.Head() != head {
		t.Errorf("tips are %s...%s, want %s...%s", tips.Base(), tips.Head(), base, head)
	}
}

func TestParseBranchTipsRefusesATipThatNamesNoCommit(t *testing.T) {
	commit := mustParseCommitSHA(t, headTipCommit)
	refused := map[string][2]domain.CommitSHA{
		"no base": {domain.CommitSHA{}, commit},
		"no head": {commit, domain.CommitSHA{}},
	}

	for name, tips := range refused {
		_, err := domain.ParseBranchTips(tips[0], tips[1])

		if !errors.Is(err, domain.ErrMalformedCommitSHA) {
			t.Errorf("%s: error is %v, want %v", name, err, domain.ErrMalformedCommitSHA)
		}
	}
}
