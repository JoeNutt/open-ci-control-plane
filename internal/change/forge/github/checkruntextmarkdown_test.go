// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"strings"
	"testing"
)

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

func TestMarkdownCodeSpanShowsFormatAndControlCharactersAsMarkers(t *testing.T) {
	marked := map[string]markdownCodeSpanCase{
		"right-to-left override":  {"a\u202Eb", "` a<U+202E>b `"},
		"left-to-right embedding": {"a\u202Ab", "` a<U+202A>b `"},
		"first strong isolate":    {"a\u2068b", "` a<U+2068>b `"},
		"pop directional isolate": {"a\u2069b", "` a<U+2069>b `"},
		"zero width space":        {"a\u200Bb", "` a<U+200B>b `"},
		"zero width joiner":       {"a\u200Db", "` a<U+200D>b `"},
		"byte order mark":         {"a\uFEFFb", "` a<U+FEFF>b `"},
		"soft hyphen":             {"a\u00ADb", "` a<U+00AD>b `"},
		"tab":                     {"a\tb", "` a<U+0009>b `"},
		"null":                    {"a\x00b", "` a<U+0000>b `"},
		"delete":                  {"a\x7Fb", "` a<U+007F>b `"},
		"C1 control":              {"a\u0090b", "` a<U+0090>b `"},
	}

	assertMarkdownCodeSpans(t, marked)
}

func TestMarkdownCodeSpanShowsAnEmptyOrBlankValueVisibly(t *testing.T) {
	blank := map[string]markdownCodeSpanCase{
		"empty":                {"", "` <empty> `"},
		"one space":            {" ", "` <U+0020> `"},
		"three spaces":         {"   ", "` <U+0020><U+0020><U+0020> `"},
		"a line feed only":     {"\n", "` <U+0020> `"},
		"spaces around a word": {" a ", "`  a  `"},
	}

	assertMarkdownCodeSpans(t, blank)
}

func TestMarkdownCodeSpanNeverEmitsAFenceLongerThanSixteen(t *testing.T) {
	for run := 1; run <= 200; run++ {
		span := markdownCodeSpanOf(strings.Repeat("`", run) + "x")

		fence := len(span) - len(strings.TrimLeft(span, "`"))
		if fence > 16 {
			t.Errorf("a run of %d emits a fence of %d", run, fence)
		}
		if run >= 16 && strings.Count(span, "<U+0060>") != run {
			t.Errorf("a run of %d shows %d backtick markers, want %d",
				run, strings.Count(span, "<U+0060>"), run)
		}
	}
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
