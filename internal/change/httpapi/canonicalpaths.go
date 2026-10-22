// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"log/slog"
	"net/http"
	"path"
	"strings"
)

const (
	repositoriesPrefix = "/v1/repositories/"
	wholePathField     = "path"
)

type canonicalRepositoryPaths struct {
	routes    http.Handler
	answering answers
	log       *slog.Logger
}

func (guard canonicalRepositoryPaths) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	if !isNonCanonicalRepositoryPath(asked.URL.Path) {
		guard.routes.ServeHTTP(answer, asked)
		return
	}
	refused := malformedPathProblem(wholePathField)
	guard.log.LogAttrs(asked.Context(), slog.LevelWarn, refusedCoverage, slog.String(reasonKey, refused.reason()))
	guard.answering.write(asked.Context(), answer, refused)
}

func isNonCanonicalRepositoryPath(asked string) bool {
	return strings.HasPrefix(asked, repositoriesPrefix) && path.Clean(asked) != asked
}
