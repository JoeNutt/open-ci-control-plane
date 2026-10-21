// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

var verbsFmtAnswersBeforeAnyMethod = []string{"%T", "%p"}

func secretHolders(t *testing.T) map[string]any {
	t.Helper()

	return map[string]any{
		"credential": forgeCredential,
		"App key":    forgeAppKey(t),
	}
}

const jwtPrefix = "eyJ"

func TestInstallationTokensHoldingATokenRevealNoSecretWhenPrintedOrLogged(t *testing.T) {
	held := tokensExchangedOnce(t)
	secrets := []string{
		grantedToken(1),
		jwtPrefix,
		forgeRSAKey().D.String(),
		string(forgeAppKeyPEM()[len(github.PKCS1PrivateKeyBlock)+16 : 120]),
	}
	holders := map[string]any{
		"tokens":        held.tokens,
		"nested tokens": struct{ tokens *github.InstallationTokens }{held.tokens},
	}

	for name, holder := range holders {
		for _, verb := range everyFormatVerb {
			mustRevealNoneOf(t, name+" under "+verb, fmt.Sprintf(verb, holder), secrets)
		}
		encoded, err := json.Marshal(holder)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		mustRevealNoneOf(t, name+" as JSON", string(encoded), secrets)
		mustRevealNoneOf(t, name+" logged", loggedAsTextAndJSON(holder), secrets)
	}
}

func loggedAsTextAndJSON(holder any) string {
	var logged bytes.Buffer
	slog.New(slog.NewTextHandler(&logged, nil)).Info("forge", "holder", holder)
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("forge", "holder", holder)
	return logged.String()
}

func mustRevealNoneOf(t *testing.T, where, printed string, secrets []string) {
	t.Helper()

	for _, secret := range secrets {
		if reveals(printed, secret) {
			t.Errorf("%s revealed a secret: %.120s", where, printed)
		}
	}
}

func TestASecretMarshalsToJSONAsTheRedactionMarker(t *testing.T) {
	for name, secret := range secretHolders(t) {
		encoded, err := json.Marshal(secret)

		if err != nil {
			t.Fatalf("marshal the %s: %v", name, err)
		}
		if string(encoded) != strconv.Quote(github.RedactionMarker) {
			t.Errorf("the %s marshals to %.80s, want %q", name, encoded, github.RedactionMarker)
		}
	}
}

func TestASecretPrintsOnlyTheRedactionMarkerUnderEveryVerb(t *testing.T) {
	for name, secret := range secretHolders(t) {
		for _, verb := range everyFormatVerb {
			if slices.Contains(verbsFmtAnswersBeforeAnyMethod, verb) {
				continue
			}
			if printed := fmt.Sprintf(verb, secret); printed != github.RedactionMarker {
				t.Errorf("the %s under %s printed %.80q, want %q", name, verb, printed, github.RedactionMarker)
			}
		}
	}
}
