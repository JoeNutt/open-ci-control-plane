// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestATokenEndpointErrorFailsTheForgeCallWithNoRequestUnauthenticated(t *testing.T) {
	clock := clockAt(mintedAt)
	endpoint := newTokenForge(t, clock)
	endpoint.refuseFromNowOn()
	settings := settingsAt(endpoint.server)
	settings.Credentials = installationTokensOn(t, endpoint.server, clock)

	_, err := sourceWith(t, settings).FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Fatalf("error is %v, want %v from the token endpoint", err, github.ErrUnexpectedStatus)
	}
}

func TestCheckRunsPresentTheMintedInstallationToken(t *testing.T) {
	clock := clockAt(mintedAt)
	endpoint := newTokenForge(t, clock)
	held := newCheckRunForge(t)
	endpoint.answeringAsTheRepository(held.server.Config.Handler)
	settings := settingsAt(endpoint.server)
	settings.Credentials = installationTokensOn(t, endpoint.server, clock)
	run := securityRun()

	err := checkRunsWith(t, settings).PublishCheckRun(
		context.Background(), locatorOf(t, "acme/payments", 412), headCommit, run)

	if err != nil {
		t.Fatalf("publish check run: %v", err)
	}
	held.mustHoldOnly(heldAs(1, run))
}

func TestAChangeSourcePresentsTheMintedInstallationToken(t *testing.T) {
	clock := clockAt(mintedAt)
	endpoint := newTokenForge(t, clock)
	endpoint.answeringAsTheRepository(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		write(t, writer, `[]`)
	}))
	settings := settingsAt(endpoint.server)
	settings.Credentials = installationTokensOn(t, endpoint.server, clock)

	_, err := sourceWith(t, settings).FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
}
