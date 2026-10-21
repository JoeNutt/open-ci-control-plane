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
}

type Binding struct {
	repository GovernedRepository
	workflow   WorkflowFile
	alwaysRun  []GoImportPath
}

func ParseBinding(stated BindingStatement) (Binding, error) {
	repository, err := ParseGovernedRepository(stated.Repository)
	if err != nil {
		return Binding{}, fmt.Errorf("binding: %w", err)
	}
	workflow, err := ParseWorkflowFile(stated.Workflow)
	if err != nil {
		return Binding{}, fmt.Errorf("binding for %s: %w", repository, err)
	}
	var alwaysRun []GoImportPath
	for _, s := range stated.AlwaysRun {
		p, err := ParseGoImportPath(s)
		if err != nil {
			return Binding{}, fmt.Errorf("binding for %s: %w", repository, err)
		}
		alwaysRun = append(alwaysRun, p)
	}
	return Binding{repository: repository, workflow: workflow, alwaysRun: alwaysRun}, nil
}

func (binding Binding) Repository() GovernedRepository { return binding.repository }

func (binding Binding) Workflow() WorkflowFile { return binding.workflow }

func (binding Binding) AlwaysRun() []GoImportPath { return slices.Clone(binding.alwaysRun) }
