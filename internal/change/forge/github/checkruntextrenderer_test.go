// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"os/exec"
	"strings"
	"testing"
)

const checkRunTextHostilePayload = " [click me](https://evil.test/login) " +
	"![t](https://evil.test/pixel.png) <img src=x onerror=alert(1)> **Approved** @octocat"

func TestCheckRunTextStaysInertThroughCmarkGFM(t *testing.T) {
	renderer := checkRunTextCmarkGFM(t)

	for _, run := range []int{1, 15, 16, 17, 79, 80, 81, 200} {
		forged := checkRunTextFixture(t)
		forged.Violations[0].Message = strings.Repeat("`", run) + checkRunTextHostilePayload
		output, err := checkRunTextFor(forged, checkRunTextTestCapacity)
		if err != nil {
			t.Fatalf("render: %v", err)
		}

		html := checkRunTextHTMLOf(t, renderer, output.text)

		for _, live := range []string{`href="https://evil.test`, `src="https://evil.test`, "<img", "<strong>Approved"} {
			if strings.Contains(html, live) {
				t.Errorf("a run of %d rendered %q live:\n%s", run, live, html)
			}
		}
	}
}

func checkRunTextCmarkGFM(t *testing.T) string {
	t.Helper()

	renderer, err := exec.LookPath("cmark-gfm")
	if err != nil {
		t.Fatalf("cmark-gfm is not on PATH; the dev shell provides it (nix develop): %v", err)
	}
	return renderer
}

func checkRunTextHTMLOf(t *testing.T, renderer, markdown string) string {
	t.Helper()

	render := exec.Command(renderer, "-e", "autolink", "-e", "table", "-e", "strikethrough", "-e", "tagfilter")
	render.Stdin = strings.NewReader(markdown)
	html, err := render.Output()
	if err != nil {
		t.Fatalf("run %s: %v", renderer, err)
	}
	return string(html)
}
