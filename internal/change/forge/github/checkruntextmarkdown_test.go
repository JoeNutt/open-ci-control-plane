// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "testing"

func TestMarkdownCodeSpanFenceOutrunsEveryBacktickRunInTheValue(t *testing.T) {
	fenced := map[string]string{
		"plain":                 "` plain `",
		"one backtick":          "`` a`b ``",
		"a run of two":          "``` a``b` ```",
		"leading and trailing":  "`` `x` ``",
		"a run of three opener": "```` ```js ````",
	}
	values := map[string]string{
		"plain":                 "plain",
		"one backtick":          "a`b",
		"a run of two":          "a``b`",
		"leading and trailing":  "`x`",
		"a run of three opener": "```js",
	}

	for form, want := range fenced {
		t.Run(form, func(t *testing.T) {
			if got := markdownCodeSpanOf(values[form]); got != want {
				t.Errorf("span is %q, want %q", got, want)
			}
		})
	}
}
