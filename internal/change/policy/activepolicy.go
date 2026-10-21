// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"errors"
	"sync/atomic"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrNoPolicyLoaded = errors.New("no policy loaded")

type AcceptedRevision struct {
	revision domain.CommitSHA
	model    domain.PolicyModel
}

func (accepted AcceptedRevision) Revision() domain.CommitSHA { return accepted.revision }

type ActivePolicy struct {
	accepted atomic.Pointer[AcceptedRevision]
}

func NewActivePolicy() *ActivePolicy { return &ActivePolicy{} }

func (active *ActivePolicy) Offer(revision domain.CommitSHA, files []domain.PolicyFile) error {
	model, err := ParseRevision(files)
	if err != nil {
		return err
	}
	active.accepted.Store(&AcceptedRevision{revision: revision, model: model})
	return nil
}

func (active *ActivePolicy) Accepted() (AcceptedRevision, error) {
	accepted := active.accepted.Load()
	if accepted == nil {
		return AcceptedRevision{}, ErrNoPolicyLoaded
	}
	return *accepted, nil
}
