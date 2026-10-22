// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ListedFile struct {
	Path  string
	Bytes int
}

type PolicyListing struct {
	files []ListedFile
}

func ParsePolicyListing(listed []ListedFile) (PolicyListing, error) {
	if breach := policyFileBoundBreachIn(listed); breach != nil {
		return PolicyListing{}, breach
	}
	return PolicyListing{files: slices.Clone(listed)}, nil
}
