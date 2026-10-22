// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type ListedFile struct {
	Path  string
	Bytes int
}

type PolicyListing struct {
	files []ListedFile
}

func ParsePolicyListing(listed []ListedFile) (PolicyListing, error) {
	if len(listed) > MaxPolicyFiles {
		return PolicyListing{}, fmt.Errorf("%w: %d exceeds %d", ErrTooManyPolicyFiles, len(listed), MaxPolicyFiles)
	}
	return PolicyListing{files: listed}, nil
}
