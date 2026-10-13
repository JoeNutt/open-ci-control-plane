// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"net/http"
)

type Settings struct {
	BaseURL          string
	Token            Credential
	Client           *http.Client
	MaxResponseBytes int
}

var ErrNoCredential = errors.New("a change source is given no forge credential")

func ParseChangeSource(settings Settings) (ChangeSource, error) {
	if settings.Token == "" {
		return ChangeSource{}, ErrNoCredential
	}
	return ChangeSource{settings: settings}, nil
}
