// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type BindingStatement struct {
	Repository string
	Workflow   string
}

type Binding struct {
	repository GovernedRepository
	workflow   WorkflowFile
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
	return Binding{repository: repository, workflow: workflow}, nil
}

func (binding Binding) Repository() GovernedRepository { return binding.repository }

func (binding Binding) Workflow() WorkflowFile { return binding.workflow }
