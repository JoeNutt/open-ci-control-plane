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
