// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type ListedFile struct {
	Path  string
	Bytes int
}

type PolicyListing struct {
	files []ListedFile
	paths []RepositoryPath
}

func ParsePolicyListing(listed []ListedFile) (PolicyListing, error) {
	if breach := policyFileBoundBreachIn(listed); breach != nil {
		return PolicyListing{}, breach
	}
	paths := make([]RepositoryPath, 0, len(listed))
	for _, file := range listed {
		path, err := ParseRepositoryPath(file.Path)
		if err != nil {
			return PolicyListing{}, err
		}
		paths = append(paths, path)
	}
	return PolicyListing{files: slices.Clone(listed), paths: paths}, nil
}

func (listing PolicyListing) Files() []ListedFile { return slices.Clone(listing.files) }

func (listing PolicyListing) Paths() []RepositoryPath { return slices.Clone(listing.paths) }
