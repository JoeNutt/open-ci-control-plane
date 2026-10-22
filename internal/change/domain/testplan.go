// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
	"strings"
)

type PlanTrigger string

const (
	PlanTriggerNone PlanTrigger = "none"
)

type TestPlan struct {
	DecisionID     DecisionID
	Repository     GovernedRepository
	Head           CommitSHA
	Workflow       WorkflowFile
	Suites         []CoverageSuite
	AlwaysRun      []GoImportPath
	FullMatrix     bool
	Trigger        PlanTrigger
	BundleRevision BundleRevision
}

type PlanBasis struct {
	Document   Document
	Coverage   RepositoryCoverage
	Binding    Binding
	DecisionID DecisionID
	Revision   BundleRevision
}

func PlanFor(basis PlanBasis) (TestPlan, error) {
	head, err := ParseCommitSHA(basis.Document.PR.HeadSHA)
	if err != nil {
		return TestPlan{}, fmt.Errorf("plan tests: %w", err)
	}
	var suites []CoverageSuite
	for _, file := range basis.Document.Diff.Files {
		path, err := ParseRepositoryPath(file.Path)
		if err != nil {
			return TestPlan{}, fmt.Errorf("plan tests: %w", err)
		}
		for _, suite := range basis.Coverage.SuitesCovering(path) {
			if !slices.Contains(suites, suite) {
				suites = append(suites, suite)
			}
		}
	}
	slices.SortFunc(suites, func(one, other CoverageSuite) int { return strings.Compare(one.String(), other.String()) })
	return TestPlan{
		DecisionID:     basis.DecisionID,
		Repository:     basis.Binding.Repository(),
		Head:           head,
		Workflow:       basis.Binding.Workflow(),
		Suites:         suites,
		AlwaysRun:      basis.Binding.AlwaysRun(),
		Trigger:        PlanTriggerNone,
		BundleRevision: basis.Revision,
	}, nil
}

func (repositoryCoverage RepositoryCoverage) SuitesCovering(path RepositoryPath) []CoverageSuite {
	var covering []CoverageSuite
	for suite, coverage := range repositoryCoverage.suites {
		if coverage.Covers(path) {
			covering = append(covering, suite)
		}
	}
	return covering
}
