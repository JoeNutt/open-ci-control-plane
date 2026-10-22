// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "net/url"

type ownedRepository interface {
	Owner() string
	Name() string
}

const urlPathSeparator = "/"

func repositoryEndpoint(repository ownedRepository) string {
	return "/repos/" + url.PathEscape(repository.Owner()) + urlPathSeparator + url.PathEscape(repository.Name())
}
