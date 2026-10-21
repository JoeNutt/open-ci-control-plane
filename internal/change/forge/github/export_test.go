// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

func RenderedForGitHub(run domain.CheckRun) (title, summary, text string, err error) {
	rendered, err := checkRunTextFor(run, MaxOutputCharacters)
	return rendered.title, rendered.summary, rendered.text, err
}

var ErrCheckRunTextUnidentified = errCheckRunTextUnidentified

var AppJWTAt = appJWTAt
