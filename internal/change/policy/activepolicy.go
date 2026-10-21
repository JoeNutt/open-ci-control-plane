// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"errors"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrNoPolicyLoaded = errors.New("no policy loaded")

type AcceptedRevision struct {
	revision domain.CommitSHA
	model    domain.PolicyModel
}

type ActivePolicy struct{}

func NewActivePolicy() *ActivePolicy { return &ActivePolicy{} }

func (active *ActivePolicy) Accepted() (AcceptedRevision, error) {
	return AcceptedRevision{}, ErrNoPolicyLoaded
}
