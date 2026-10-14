// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"regexp"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var checkRunTextInjectionPayloads = map[string]string{
	"a four-space code block":  "    indented as code",
	"an email autolink":        "owner@example.test",
	"a bare www autolink":      "www.evil.test/login",
	"an entity spoof":          "&#8238;gpj.exe",
	"a right-to-left override": "report\u202Egpj.exe",
	"a zero-width joiner":      "secu\u200Drity",
	"a zero-width space":       "secu\u200Brity",
	"a forged heading":         "fine\n\n# Passed",
	"a forged list":            "fine\r\n- [x] approved",
	"a javascript link":        "[docs](javascript:alert(1))",
	"raw HTML":                 "<img src=x onerror=alert(1)>",
	"a forged table":           "| a | b |\n|---|---|",
	"a fence opener":           "```js",
	"emphasis":                 "*bold* _em_ ~~strike~~",
	"a mention":                "@acme/security-platform",
	"a backslash escape":       "\\*not emphasis\\*",
	"a line separator":         "fine\u2028# Passed",
}

var checkRunTextLinePrefixes = []string{
	"**Rule** ", "Message: ", "- Owning team: ", "- Documentation: ", "and ",
}

func TestCheckRunTextKeepsEveryTenantFieldInsideItsSpan(t *testing.T) {
	fields := map[string]struct {
		forge         func(*domain.CheckRunViolation, string)
		before, after string
	}{
		"rule_id": {func(violation *domain.CheckRunViolation, payload string) { violation.RuleID = payload },
			"**Rule** ", " \u2014 blocking"},
		"message": {func(violation *domain.CheckRunViolation, payload string) { violation.Message = payload },
			"Message: ", ""},
		"owning_team": {func(violation *domain.CheckRunViolation, payload string) { violation.OwningTeam = payload },
			"- Owning team: ", ""},
	}

	for field, placed := range fields {
		for form, payload := range checkRunTextInjectionPayloads {
			t.Run(field+"/"+form, func(t *testing.T) {
				run := checkRunTextFixture(t)
				placed.forge(&run.Violations[0], payload)

				output, err := checkRunTextFor(run, checkRunTextTestCapacity)
				if err != nil {
					t.Fatalf("render: %v", err)
				}

				assertCheckRunTextLinesAreTheRenderers(t, output.text)
				segment := checkRunTextSegmentBetween(output.text, placed.before, placed.after)
				if !checkRunTextIsFencedAgainst(segment, payload) {
					t.Errorf("the %s is not inside a fence it cannot close: %q", field, segment)
				}
				if strings.ContainsFunc(strings.ReplaceAll(output.text, "\n", ""), markdownIsHidden) {
					t.Errorf("a hidden character reached the output:\n%q", output.text)
				}
			})
		}
	}
}

func assertCheckRunTextLinesAreTheRenderers(t *testing.T, text string) {
	t.Helper()

	for _, line := range strings.Split(text, "\n") {
		if line != "" && !checkRunTextStartsWithAnyOf(line, checkRunTextLinePrefixes) {
			t.Errorf("a line the renderer did not write: %q", line)
		}
	}
}

func checkRunTextSegmentBetween(text, before, after string) string {
	for _, line := range strings.Split(text, "\n") {
		if segment, found := strings.CutPrefix(line, before); found {
			return strings.TrimSuffix(segment, after)
		}
	}
	return ""
}

var checkRunTextBacktickRuns = regexp.MustCompile("`+")

func checkRunTextIsFencedAgainst(segment, payload string) bool {
	fence := checkRunTextBacktickRuns.FindString(segment)
	if !strings.HasPrefix(segment, fence+" ") || !strings.HasSuffix(segment, " "+fence) {
		return false
	}
	for _, run := range checkRunTextBacktickRuns.FindAllString(payload, -1) {
		if len(run) >= len(fence) {
			return false
		}
	}
	return len(fence) > 0 && len(segment) >= 2*len(fence)+2
}

func checkRunTextStartsWithAnyOf(line string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
