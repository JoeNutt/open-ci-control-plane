// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import "errors"

var ErrNoPolicyLoaded = errors.New("no policy loaded")

type AcceptedRevision struct{}

type ActivePolicy struct{}

func NewActivePolicy() *ActivePolicy { return &ActivePolicy{} }

func (active *ActivePolicy) Accepted() (AcceptedRevision, error) {
	return AcceptedRevision{}, ErrNoPolicyLoaded
}
