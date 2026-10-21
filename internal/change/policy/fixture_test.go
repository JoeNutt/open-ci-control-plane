// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"os"
	"strings"
	"testing"
)

const (
	fixtureDirectory = "testdata/"
	liveCatalogFile  = "catalog/clean-architecture.yaml"
	liveBindingFile  = "bindings/JoeNutt/open-ci-control-plane.yaml"
	liveCatalogPath  = fixtureDirectory + liveCatalogFile
	liveBindingPath  = fixtureDirectory + liveBindingFile
	liveRepository   = "JoeNutt/open-ci-control-plane"
	liveWorkflow     = "open-ci-tests.yml"
	liveWorkflowLine = "  workflow: " + liveWorkflow
)

func fixtureWith(t *testing.T, path, written, replacement string) string {
	t.Helper()

	live := string(fixture(t, path))
	if !strings.Contains(live, written) {
		t.Fatalf("%s does not contain %q, so the replacement tests nothing", path, written)
	}
	return strings.Replace(live, written, replacement, 1)
}

func fixture(t *testing.T, path string) []byte {
	t.Helper()

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return written
}
