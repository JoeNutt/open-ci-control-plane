// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
	"strings"
)

func testPackageOf(file RepositoryPath, roots []ImportRoot) (GoImportPath, error) {
	segments := file.Segments()
	directory := segments[:len(segments)-1]
	best := -1
	var bestPath string
	for _, root := range roots {
		var rootSegments []string
		if root.directory.String() != "" {
			rootSegments = strings.Split(root.directory.String(), "/")
		}
		if len(rootSegments) > len(directory) || !slices.Equal(directory[:len(rootSegments)], rootSegments) {
			continue
		}
		if len(rootSegments) > best {
			best = len(rootSegments)
			rest := directory[len(rootSegments):]
			bestPath = root.prefix.String()
			if len(rest) > 0 {
				bestPath += "/" + strings.Join(rest, "/")
			}
		}
	}
	if best < 0 {
		return GoImportPath{}, errors.New("no root")
	}
	return ParseGoImportPath(bestPath)
}
