// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const FilesPerPage = 100

// The files endpoint of a pull request serves at most 3,000 files, so 30 pages of
// FilesPerPage is every page it can offer: https://docs.github.com/en/rest/pulls/pulls
const maxFilePages = 30

func (source ChangeSource) changedFilesAcrossPages(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]changedFile, error) {
	return acrossPages[changedFile](ctx, source.forge, paging{
		endpointAt: func(page int) string { return changedFilesEndpoint(locator, page) },
		perPage:    FilesPerPage,
		maxPages:   maxFilePages,
	})
}

func changedFilesEndpoint(locator domain.PullRequestLocator, page int) string {
	return fmt.Sprintf("%s/files?per_page=%d&page=%d", pullRequestEndpoint(locator), FilesPerPage, page)
}

type changedFile struct {
	Filename         string  `json:"filename"`
	PreviousFilename string  `json:"previous_filename"`
	Status           string  `json:"status"`
	Additions        int     `json:"additions"`
	Deletions        int     `json:"deletions"`
	Patch            *string `json:"patch"`
}

func filesOf(listed []changedFile) ([]domain.StatedFile, error) {
	files := make([]domain.StatedFile, 0, len(listed))
	for _, listing := range listed {
		stated, err := statedFileOf(listing)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", listing.Filename, err)
		}
		files = append(files, stated)
	}
	return files, nil
}

func statedFileOf(listing changedFile) (domain.StatedFile, error) {
	action, err := domain.ParseFileAction(listing.Status)
	if err != nil {
		return domain.StatedFile{}, err
	}
	patch, isPatchStated := listing.statedPatch()
	return domain.StatedFile{
		File: domain.File{
			Path:      listing.Filename,
			Action:    action,
			Additions: listing.Additions,
			Deletions: listing.Deletions,
		},
		PreviousPath:  listing.PreviousFilename,
		Patch:         patch,
		IsPatchStated: isPatchStated,
	}, nil
}

func (listing changedFile) statedPatch() (string, bool) {
	if listing.Patch == nil {
		return "", false
	}
	return *listing.Patch, true
}
