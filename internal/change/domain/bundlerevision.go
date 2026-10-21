// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type BundleRevision struct {
	value string
}

func ParseBundleRevision(stated string) (BundleRevision, error) {
	return BundleRevision{value: stated}, nil
}

func (revision BundleRevision) String() string { return revision.value }
