// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const FilesPerPage = 100

// The files endpoint of a pull request serves at most 3,000 files, so 30 pages of
// FilesPerPage is every page it can offer: https://docs.github.com/en/rest/pulls/pulls
const maxFilePages = 30

var ErrTooManyPages = errors.New("the forge offered more pages than its file listing can serve")

func (source ChangeSource) changedFilesAcrossPages(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]changedFile, error) {
	var listed []changedFile
	for page := 1; page <= maxFilePages; page++ {
		onPage, err := jsonAt[[]changedFile](
			ctx, source, changedFilesEndpoint(locator, page))
		if err != nil {
			return nil, err
		}
		listed = append(listed, onPage...)
		if len(onPage) < FilesPerPage {
			return listed, nil
		}
	}
	return nil, fmt.Errorf("%w: %d pages of %d", ErrTooManyPages, maxFilePages, FilesPerPage)
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
