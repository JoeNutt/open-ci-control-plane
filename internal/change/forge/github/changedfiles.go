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
	return acrossPages[changedFile](ctx, source, paging{
		endpointAt: func(page int) string { return changedFilesEndpoint(locator, page) },
		perPage:    FilesPerPage,
		maxPages:   maxFilePages,
	})
}

func changedFilesEndpoint(locator domain.PullRequestLocator, page int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/files?per_page=%d&page=%d",
		locator.Owner(), locator.Name(), locator.Number(), FilesPerPage, page)
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
