// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func AuthorizationOf(ctx context.Context, tokens *InstallationTokens) (string, error) {
	credential, err := tokens.authorization(ctx)
	if err != nil {
		return "", err
	}
	return credential.bearer(), nil
}

func RenderedForGitHub(run domain.CheckRun) (title, summary, text string, err error) {
	rendered, err := checkRunTextFor(run, MaxOutputCharacters)
	return rendered.title, rendered.summary, rendered.text, err
}

var ErrCheckRunTextUnidentified = errCheckRunTextUnidentified

var AppJWTAt = appJWTAt

const PKCS1PrivateKeyBlock = pkcs1PrivateKeyBlock

const RedactionMarker = redactionMarker

var ErrNotAFile = errNotAFile

var ErrContentsNotServed = errContentsNotServed

var ErrContentsSizeMismatch = errContentsSizeMismatch

func CheckRunsPausingWith(runs CheckRuns, pause func(context.Context, time.Duration) error) CheckRuns {
	runs.pause = pause
	return runs
}
