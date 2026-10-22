// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
)

func TestAnUploadFromAnotherRepositoryIsAnswered403(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("authorise: %w", application.ErrForeignRepository))

	if refused.answer.status() != http.StatusForbidden {
		t.Errorf("answered %#v, want %d", refused.answer, http.StatusForbidden)
	}
}
