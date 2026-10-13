// SPDX-License-Identifier: GPL-3.0-or-later

// Package github answers for a forge over the GitHub REST API.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const FilesPerPage = 100

var ErrUnexpectedStatus = errors.New("the forge answered with an unexpected status")

var reportedByStatus = map[int]error{
	http.StatusNotFound: domain.ErrNoSuchPullRequest,
}

func failureIn(response *http.Response) error {
	if response.StatusCode == http.StatusOK {
		return nil
	}
	reported, mapped := reportedByStatus[response.StatusCode]
	if !mapped {
		reported = ErrUnexpectedStatus
	}
	return fmt.Errorf("%w: %s", reported, response.Status)
}

type ChangeSource struct {
	BaseURL string
	Client  *http.Client
}

func (source ChangeSource) FilesChanged(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	listed, err := source.changedFilesAcrossPages(ctx, locator)
	if err != nil {
		return nil, fmt.Errorf("list the changed files of %s: %w", locator, err)
	}
	files, err := filesOf(listed)
	if err != nil {
		return nil, fmt.Errorf("read the changed files of %s: %w", locator, err)
	}
	return files, nil
}

func (source ChangeSource) changedFilesAcrossPages(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]changedFile, error) {
	var listed []changedFile
	for page := 1; ; page++ {
		onPage, err := jsonAt[[]changedFile](
			ctx, source.Client, source.changedFilesAddress(locator, page))
		if err != nil {
			return nil, err
		}
		listed = append(listed, onPage...)
		if len(onPage) < FilesPerPage {
			return listed, nil
		}
	}
}

func (source ChangeSource) changedFilesAddress(
	locator domain.PullRequestLocator, page int,
) string {
	return fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=%d&page=%d",
		source.BaseURL, locator.Owner(), locator.Name(), locator.Number(), FilesPerPage, page)
}

type changedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

func filesOf(listed []changedFile) ([]domain.File, error) {
	files := make([]domain.File, 0, len(listed))
	for _, listing := range listed {
		action, err := domain.ParseFileAction(listing.Status)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", listing.Filename, err)
		}
		files = append(files, domain.File{
			Path:       listing.Filename,
			Action:     action,
			Additions:  listing.Additions,
			Deletions:  listing.Deletions,
			PatchBytes: len(listing.Patch),
		})
	}
	return files, nil
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

	if failure := failureIn(response); failure != nil {
		return payload, failure
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return payload, fmt.Errorf("decode: %w", err)
	}
	return payload, nil
}
