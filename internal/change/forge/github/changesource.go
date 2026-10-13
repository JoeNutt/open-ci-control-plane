// SPDX-License-Identifier: GPL-3.0-or-later

// Package github answers for a forge over the GitHub REST API.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeSource struct {
	BaseURL string
	Client  *http.Client
}

type changedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

func (source ChangeSource) FilesChanged(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	address := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files",
		source.BaseURL, locator.Owner(), locator.Name(), locator.Number())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("request changed files of %s: %w", locator, err)
	}
	response, err := source.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("read changed files of %s: %w", locator, err)
	}
	defer response.Body.Close()

	var listed []changedFile
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		return nil, fmt.Errorf("decode changed files of %s: %w", locator, err)
	}
	files := make([]domain.File, 0, len(listed))
	for _, file := range listed {
		files = append(files, domain.File{
			Path:       file.Filename,
			Action:     domain.FileAction(file.Status),
			Additions:  file.Additions,
			Deletions:  file.Deletions,
			PatchBytes: len(file.Patch),
		})
	}
	return files, nil
}
