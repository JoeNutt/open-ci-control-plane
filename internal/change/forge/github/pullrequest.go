// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func pullRequestEndpoint(locator domain.PullRequestLocator) string {
	return fmt.Sprintf("%s/pulls/%d", repositoryEndpoint(locator), locator.Number())
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
	User              account `json:"user"`
	AuthorAssociation string  `json:"author_association"`
	Labels            []label `json:"labels"`
}

type account struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

type label struct {
	Name string `json:"name"`
}

func pullRequestOf(
	stated pullRequest, locator domain.PullRequestLocator,
) (domain.PullRequest, error) {
	author, err := authorOf(stated.User, stated.AuthorAssociation)
	if err != nil {
		return domain.PullRequest{}, err
	}
	return domain.PullRequest{
		Number:     stated.Number,
		Repository: locator.Repository(),
		BaseRef:    stated.Base.Ref,
		HeadSHA:    stated.Head.SHA,
		Draft:      stated.Draft,
		Author:     author,
		Labels:     labelsOf(stated.Labels),
	}, nil
}

func authorOf(opened account, association string) (domain.Author, error) {
	isBot, classified := isBotByUserType[opened.Type]
	if !classified {
		return domain.Author{}, fmt.Errorf("%w: %q opened by %q",
			ErrUnknownUserType, opened.Type, opened.Login)
	}
	declared, err := domain.ParseAuthorAssociation(association)
	if err != nil {
		return domain.Author{}, fmt.Errorf("the association of %q: %w", opened.Login, err)
	}
	return domain.Author{Login: opened.Login, Association: declared, IsBot: isBot}, nil
}

func labelsOf(worn []label) domain.Declared[string] {
	labels := make(domain.Declared[string], 0, len(worn))
	for _, label := range worn {
		labels = append(labels, label.Name)
	}
	return labels
}

var ErrUnknownUserType = errors.New("the forge names an account type with no bot classification")

var isBotByUserType = map[string]bool{
	"User":         false,
	"Bot":          true,
	"Organization": false,
}
