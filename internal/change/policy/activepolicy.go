// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrNoPolicyLoaded = errors.New("no policy loaded")

type ActivePolicy struct {
	log      *slog.Logger
	accepted atomic.Pointer[AcceptedRevision]
}

func NewActivePolicy(log *slog.Logger) *ActivePolicy { return &ActivePolicy{log: log} }

func (active *ActivePolicy) Offer(revision domain.CommitSHA, files []domain.PolicyFile) error {
	model, err := ParseRevision(files)
	if err != nil {
		active.log.Error("policy revision refused", "revision", revision.String(), "reason", err.Error())
		return fmt.Errorf("candidate at %s: %w", revision, err)
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

type AcceptedRevision struct {
	revision domain.CommitSHA
	model    domain.PolicyModel
}

func (accepted AcceptedRevision) Revision() domain.CommitSHA { return accepted.revision }
