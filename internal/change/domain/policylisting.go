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

var boundsOfAPolicyListing = []func([]ListedFile) error{
	policyFileCountBreachIn,
	malformedPolicyFilePathIn,
	oversizedPolicyFileIn,
}

func ParsePolicyListing(listed []ListedFile) (PolicyListing, error) {
	for _, breachIn := range boundsOfAPolicyListing {
		if breach := breachIn(listed); breach != nil {
			return PolicyListing{}, breach
		}
	}
	return PolicyListing{files: slices.Clone(listed)}, nil
}
