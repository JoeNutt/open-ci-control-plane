// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func pullRequestEndpoint(locator domain.PullRequestLocator) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d",
		locator.Owner(), locator.Name(), locator.Number())
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
	Labels []label `json:"labels"`
}

type label struct {
	Name string `json:"name"`
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
		Labels: labelsOf(stated.Labels),
	}, nil
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
