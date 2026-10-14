// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "testing"

func TestMarkdownCodeSpanFenceOutrunsEveryBacktickRunInTheValue(t *testing.T) {
	fenced := map[string]markdownCodeSpanCase{
		"plain":                 {"plain", "` plain `"},
		"one backtick":          {"a`b", "`` a`b ``"},
		"a run of two":          {"a``b`", "``` a``b` ```"},
		"leading and trailing":  {"`x`", "`` `x` ``"},
		"a run of three opener": {"```js", "```` ```js ````"},
	}

	assertMarkdownCodeSpans(t, fenced)
}

func TestMarkdownCodeSpanTurnsEveryLineTerminatorIntoASpace(t *testing.T) {
	terminated := map[string]markdownCodeSpanCase{
		"line feed":            {"a\nb", "` a b `"},
		"carriage return":      {"a\rb", "` a b `"},
		"CR LF":                {"a\r\nb", "` a b `"},
		"vertical tab":         {"a\vb", "` a b `"},
		"form feed":            {"a\fb", "` a b `"},
		"next line":            {"a\u0085b", "` a b `"},
		"line separator":       {"a\u2028b", "` a b `"},
		"paragraph separator":  {"a\u2029b", "` a b `"},
		"a forged heading too": {"x\n\n# Passed", "` x  # Passed `"},
	}

	assertMarkdownCodeSpans(t, terminated)
}

type markdownCodeSpanCase struct{ value, span string }

func assertMarkdownCodeSpans(t *testing.T, cases map[string]markdownCodeSpanCase) {
	t.Helper()

	for form, want := range cases {
		t.Run(form, func(t *testing.T) {
			if got := markdownCodeSpanOf(want.value); got != want.span {
				t.Errorf("span is %q, want %q", got, want.span)
			}
		})
	}
}
