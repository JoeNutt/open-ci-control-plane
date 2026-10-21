// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"io"
	"net/http"
)

func Routes() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"field":"X-GitHub-Event"}`)
	})
}
