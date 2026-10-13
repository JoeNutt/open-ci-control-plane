// SPDX-License-Identifier: GPL-3.0-or-later

// Package github answers for a forge over the GitHub REST API.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeSource struct {
	BaseURL          string
	Client           *http.Client
	MaxResponseBytes int
}

func (source ChangeSource) PullRequestAt(
	ctx context.Context, locator domain.PullRequestLocator,
) (domain.PullRequest, error) {
	stated, err := jsonAt[pullRequest](ctx, source, pullRequestEndpoint(locator))
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("read the pull request %s: %w", locator, err)
	}
	return pullRequestOf(stated, locator)
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

func pullRequestEndpoint(locator domain.PullRequestLocator) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d",
		locator.Owner(), locator.Name(), locator.Number())
}

func pullRequestOf(
	stated pullRequest, locator domain.PullRequestLocator,
) (domain.PullRequest, error) {
	isBot, classified := isBotByUserType[stated.User.Type]
	if !classified {
		return domain.PullRequest{}, fmt.Errorf("%w: %q opened by %q",
			ErrUnknownUserType, stated.User.Type, stated.User.Login)
	}
	return domain.PullRequest{
		Number:     stated.Number,
		Repository: locator.Repository(),
		BaseRef:    stated.Base.Ref,
		HeadSHA:    stated.Head.SHA,
		Draft:      stated.Draft,
		Author: domain.Author{
			Login: stated.User.Login,
			IsBot: isBot,
		},
	}, nil
}

var ErrUnknownUserType = errors.New("the forge names an account type with no bot classification")

var isBotByUserType = map[string]bool{
	"User":         false,
	"Bot":          true,
	"Organization": false,
}

type pullRequest struct {
	Number int  `json:"number"`
	Draft  bool `json:"draft"`
	Base   struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		SHA string `json:"sha"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}

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
