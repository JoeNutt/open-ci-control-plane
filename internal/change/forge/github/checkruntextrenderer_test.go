// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"os/exec"
	"path/filepath"
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

	if onPath, err := exec.LookPath("cmark-gfm"); err == nil {
		return onPath
	}
	inStore, err := filepath.Glob("/nix/store/*-cmark-gfm-*/bin/cmark-gfm")
	if err != nil || len(inStore) == 0 {
		t.Skip("cmark-gfm is neither on PATH nor in /nix/store, so GitHub's renderer cannot be exercised here")
	}
	return inStore[0]
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
