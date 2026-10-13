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

func (source ChangeSource) FilesChanged(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	listed, err := jsonAt[[]changedFile](ctx, source.Client, source.changedFilesAddress(locator))
	if err != nil {
		return nil, fmt.Errorf("changed files of %s: %w", locator, err)
	}
	return filesOf(listed), nil
}

func (source ChangeSource) changedFilesAddress(locator domain.PullRequestLocator) string {
	return fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files",
		source.BaseURL, locator.Owner(), locator.Name(), locator.Number())
}

type changedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

func filesOf(listed []changedFile) []domain.File {
	files := make([]domain.File, 0, len(listed))
	for _, listing := range listed {
		files = append(files, domain.File{
			Path:       listing.Filename,
			Action:     domain.FileAction(listing.Status),
			Additions:  listing.Additions,
			Deletions:  listing.Deletions,
			PatchBytes: len(listing.Patch),
		})
	}
	return files
}

func jsonAt[Payload any](
	ctx context.Context, client *http.Client, address string,
) (Payload, error) {
	var payload Payload
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return payload, fmt.Errorf("request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return payload, fmt.Errorf("read: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return payload, fmt.Errorf("decode: %w", err)
	}
	return payload, nil
}
