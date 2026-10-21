// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"encoding/json"
	"io"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type payload struct {
	Installation *statedInstallation `json:"installation"`
	Action       string              `json:"action"`
	Number       int                 `json:"number"`
	Repository   statedRepository    `json:"repository"`
}

type statedRepository struct {
	FullName string `json:"full_name"`
}

type statedInstallation struct {
	ID *int64 `json:"id"`
}

func payloadOf(body io.Reader) (payload, error) {
	var stated payload
	err := json.NewDecoder(body).Decode(&stated)
	return stated, err
}

func (stated payload) installationID() (int64, bool) {
	if stated.Installation == nil || stated.Installation.ID == nil {
		return 0, false
	}
	return *stated.Installation.ID, true
}

func (stated payload) pullRequest() (domain.PullRequestLocator, error) {
	return domain.ParsePullRequestLocator(stated.Repository.FullName, stated.Number)
}
