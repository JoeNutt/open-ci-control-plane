// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
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
