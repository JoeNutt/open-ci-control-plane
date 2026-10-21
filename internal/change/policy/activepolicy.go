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

var ErrNoPolicyLog = errors.New("active policy has no log to record revisions in")

const (
	acceptedRevisionMessage = "policy revision accepted"
	refusedRevisionMessage  = "policy revision refused"
	revisionLogKey          = "revision"
	reasonLogKey            = "reason"
)

type ActivePolicy struct {
	log      *slog.Logger
	accepted atomic.Pointer[AcceptedRevision]
}

func NewActivePolicy(log *slog.Logger) *ActivePolicy { return &ActivePolicy{log: log} }

func ParseActivePolicy(log *slog.Logger) (*ActivePolicy, error) {
	if log == nil {
		return nil, ErrNoPolicyLog
	}
	return &ActivePolicy{log: log}, nil
}

func (active *ActivePolicy) Offer(revision domain.CommitSHA, files []domain.PolicyFile) error {
	model, err := ParseRevision(files)
	if err != nil {
		active.log.Error(refusedRevisionMessage, revisionLogKey, revision.String(), reasonLogKey, err.Error())
		return fmt.Errorf("candidate at %s: %w", revision, err)
	}
	active.accepted.Store(&AcceptedRevision{revision: revision, model: model})
	active.log.Info(acceptedRevisionMessage, revisionLogKey, revision.String())
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

func (accepted AcceptedRevision) Model() domain.PolicyModel { return accepted.model }
