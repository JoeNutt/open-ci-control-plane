// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type BindingStatement struct {
	Repository string
	Workflow   string
	AlwaysRun  []string
	Selection  string
	Policies   []BoundPolicyStatement
}

type Binding struct {
	repository GovernedRepository
	workflow   WorkflowFile
	alwaysRun  []GoImportPath
	selection  TestMode
	policies   []BoundPolicy
}

func ParseBinding(stated BindingStatement) (Binding, error) {
	if breach := bindingBoundBreachIn(stated); breach != nil {
		return Binding{}, fmt.Errorf("binding: %w", breach)
	}
	repository, err := ParseGovernedRepository(stated.Repository)
	if err != nil {
		return Binding{}, fmt.Errorf("binding: %w", err)
	}
	binding, err := parseBindingFor(repository, stated)
	if err != nil {
		return Binding{}, fmt.Errorf("binding for %s: %w", repository, err)
	}
	return binding, nil
}

func (binding Binding) Repository() GovernedRepository { return binding.repository }

func (binding Binding) Workflow() WorkflowFile { return binding.workflow }

func (binding Binding) AlwaysRun() []GoImportPath { return slices.Clone(binding.alwaysRun) }

func (binding Binding) SelectionMode() TestMode { return binding.selection }

func (binding Binding) Policies() []BoundPolicy { return slices.Clone(binding.policies) }

func parseBindingFor(repository GovernedRepository, stated BindingStatement) (Binding, error) {
	workflow, err := ParseWorkflowFile(stated.Workflow)
	if err != nil {
		return Binding{}, err
	}
	alwaysRun, err := goImportPathsOf(stated.AlwaysRun)
	if err != nil {
		return Binding{}, fmt.Errorf("always-run suite: %w", err)
	}
	policies, err := boundPoliciesOf(stated.Policies)
	if err != nil {
		return Binding{}, err
	}
	selection, err := modeOf(stated.Selection, TestModeShadow)
	if err != nil {
		return Binding{}, fmt.Errorf("selection mode: %w", err)
	}
	return Binding{repository: repository, workflow: workflow, alwaysRun: alwaysRun, selection: selection, policies: policies}, nil
}

func modeOf(stated string, unstated TestMode) (TestMode, error) {
	if stated == "" {
		return unstated, nil
	}
	return ParseTestMode(stated)
}

func goImportPathsOf(stated []string) ([]GoImportPath, error) {
	paths := make([]GoImportPath, 0, len(stated))
	for _, written := range stated {
		path, err := ParseGoImportPath(written)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
