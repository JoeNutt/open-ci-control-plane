// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"net/http"
	"testing"
)

func TestAChangeSourcePresentsTheMintedInstallationToken(t *testing.T) {
	clock := clockAt(mintedAt)
	endpoint := newTokenForge(t, clock)
	endpoint.answeringAsTheRepository(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		write(t, writer, `[]`)
	}))
	settings := settingsAt(endpoint.server)
	settings.Token = installationTokensOn(t, endpoint.server, clock)

	_, err := sourceWith(t, settings).FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
}
