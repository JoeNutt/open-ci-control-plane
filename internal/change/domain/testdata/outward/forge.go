// SPDX-License-Identifier: GPL-3.0-or-later

// Package outward is the architecture test's denial fixture. Under testdata, so never built.
package outward

import (
	_ "github.com/gin-gonic/gin"
	_ "github.com/google/go-github/v60/github"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/open-policy-agent/opa/v1/sdk"
)
