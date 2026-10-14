// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "testing"

func TestMarkdownCodeSpanFenceOutrunsEveryBacktickRunInTheValue(t *testing.T) {
	fenced := map[string]struct{ value, span string }{
		"plain":                 {"plain", "` plain `"},
		"one backtick":          {"a`b", "`` a`b ``"},
		"a run of two":          {"a``b`", "``` a``b` ```"},
		"leading and trailing":  {"`x`", "`` `x` ``"},
		"a run of three opener": {"```js", "```` ```js ````"},
	}

	for form, want := range fenced {
		t.Run(form, func(t *testing.T) {
			if got := markdownCodeSpanOf(want.value); got != want.span {
				t.Errorf("span is %q, want %q", got, want.span)
			}
		})
	}
}

func TestMarkdownCodeSpanTurnsEveryLineTerminatorIntoASpace(t *testing.T) {
	terminated := map[string]struct{ value, span string }{
		"line feed":            {"a\nb", "` a b `"},
		"carriage return":      {"a\rb", "` a b `"},
		"CR LF":                {"a\r\nb", "` a b `"},
		"vertical tab":         {"a\vb", "` a b `"},
		"form feed":            {"a\fb", "` a b `"},
		"next line":            {"a\u0085b", "` a b `"},
		"line separator":       {"a b", "` a b `"},
		"paragraph separator":  {"a b", "` a b `"},
		"a forged heading too": {"x\n\n# Passed", "` x  # Passed `"},
	}

	for form, want := range terminated {
		t.Run(form, func(t *testing.T) {
			if got := markdownCodeSpanOf(want.value); got != want.span {
				t.Errorf("span is %q, want %q", got, want.span)
			}
		})
	}
}
