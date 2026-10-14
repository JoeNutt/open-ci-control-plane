// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"net/url"
	"strings"
	"testing"
)

func TestCheckRunTextLinksOnlyAPlainHTTPSDocumentationURL(t *testing.T) {
	linked := []string{
		"https://example.test/policy/AUTHZ-1",
		"https://example.test:443/policy/AUTHZ-1",
		"https://example.test/a%20b?q=1#section-2",
		"https://example.test/" + strings.Repeat("a", 2048-len("https://example.test/")),
	}

	for _, docsURL := range linked {
		t.Run(docsURL[:min(len(docsURL), 60)], func(t *testing.T) {
			text := checkRunTextWithDocsURL(t, docsURL)

			if line := "- Documentation: <" + docsURL + ">\n"; !strings.Contains(text, line) {
				t.Errorf("text does not link %q:\n%s", docsURL, text)
			}
		})
	}
}

func TestCheckRunTextShowsAnyOtherDocumentationURLAsTextNotALink(t *testing.T) {
	refused := map[string]string{
		"a script":                 "javascript:alert(1)",
		"inline data":              "data:text/html,<script>alert(1)</script>",
		"plain http":               "http://example.test/policy/AUTHZ-1",
		"upper-case scheme":        "HTTPS://example.test/policy/AUTHZ-1",
		"no scheme":                "//example.test/policy/AUTHZ-1",
		"no host":                  "https:///policy/AUTHZ-1",
		"userinfo":                 "https://user:secret@example.test/policy",
		"a lookalike userinfo":     "https://docs.example.test@evil.test/policy",
		"another port":             "https://example.test:8443/policy",
		"an entity spoof":          "https://example.test/&#8238;gpj.exe",
		"an ampersand":             "https://example.test/?a=1&b=2",
		"a semicolon":              "https://example.test/a;b",
		"parentheses":              "https://example.test/(x)",
		"a space":                  "https://example.test/a b",
		"an angle bracket":         "https://example.test/a>b",
		"non-ASCII host":           "https://ex\u00e4mple.test/policy",
		"a right-to-left override": "https://example.test/\u202Egpj.exe",
		"a forged line":            "https://example.test/a\n# Passed",
		"an email autolink":        "mailto:owner@example.test",
		"too long":                 "https://example.test/" + strings.Repeat("a", 2049-len("https://example.test/")),
		"empty":                    "",
		"an empty host and a port": "https://:443/x",
		"an empty host and colon":  "https://:/x",
		"a lone dot":               "https://.",
		"a lone hyphen":            "https://-/",
		"an IPv4 literal":          "https://10.0.0.1/policy",
		"an IPv6 literal":          "https://[::1]/policy",
		"localhost":                "https://localhost/policy",
		"a single-label host":      "https://intranet/policy",
		"a label ending in hyphen": "https://docs-.example.test/policy",
		"a label opening a hyphen": "https://-docs.example.test/policy",
		"an empty label":           "https://docs..example.test/policy",
		"a leading dot":            "https://.example.test/policy",
		"a trailing dot":           "https://example.test./policy",
		"an all-numeric top label": "https://example.123/policy",
	}

	for form, docsURL := range refused {
		t.Run(form, func(t *testing.T) {
			text := checkRunTextWithDocsURL(t, docsURL)

			shown := "- Documentation: " + markdownCodeSpanOf(docsURL) + " (not a link)\n"
			if !strings.Contains(text, shown) {
				t.Errorf("text does not show %q as text:\n%s", docsURL, text)
			}
			if strings.Contains(text, "<"+docsURL+">") {
				t.Errorf("text links %q:\n%s", docsURL, text)
			}
		})
	}
}

func TestCheckRunTextLinkOfReturnsTheParsedLinkOrNone(t *testing.T) {
	link, isLink := checkRunTextLinkOf("https://example.test/policy/AUTHZ-1")
	if !isLink || link.address != "https://example.test/policy/AUTHZ-1" {
		t.Errorf("got %+v, %v; want the link to https://example.test/policy/AUTHZ-1", link, isLink)
	}

	if refused, isLink := checkRunTextLinkOf("javascript:alert(1)"); isLink || refused != (checkRunTextLink{}) {
		t.Errorf("javascript: gave %+v, %v; want no link", refused, isLink)
	}
}

func TestCheckRunTextLinkHostRefusesUserinfoOnItsOwn(t *testing.T) {
	parsed, err := url.Parse("https://user@docs.example.test/policy")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if checkRunTextIsLinkHost(parsed) {
		t.Error("a host with userinfo was accepted")
	}
}

func TestCheckRunTextLinkAlphabetOnItsOwn(t *testing.T) {
	alphabet := map[string]bool{
		"https://example.test/~owner":  true,
		"https://example.test/a_rule":  true,
		"https://example.test/a@b":     false,
		"https://example.test/[a]":     false,
		"https://example.test/a%20b?q": true,
	}

	for stated, accepted := range alphabet {
		t.Run(stated, func(t *testing.T) {
			if got := checkRunTextHasOnlyLinkCharacters(stated); got != accepted {
				t.Errorf("accepted %v, want %v", got, accepted)
			}
		})
	}
}

func checkRunTextWithDocsURL(t *testing.T, docsURL string) string {
	t.Helper()

	run := checkRunTextFixture(t)
	run.Violations[0].DocsURL = docsURL
	output, err := checkRunTextFor(run, checkRunTextTestCapacity)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return output.text
}
