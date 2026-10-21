// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrMissingBundleRevision = errors.New("decision carries no bundle revision")

type BundleRevision struct {
	value string
}

func ParseBundleRevision(stated string) (BundleRevision, error) {
	if stated == "" {
		return BundleRevision{}, ErrMissingBundleRevision
	}
	return BundleRevision{value: stated}, nil
}

func (revision BundleRevision) String() string { return revision.value }
