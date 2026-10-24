// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type TestExtractor interface {
	DeclaresTests(path string) bool
	TestFunctionsIn(source BoundedSource) ([]TestFunction, error)
}
