// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"path/filepath"
)

var errNotAbsolutePath = errors.New("the path is not absolute")

func historyFileIn(stated statedItems) (string, error) {
	path, err := valueOf[string](stated, historyFileItem)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		return "", historyFileItem.invalid(errNotAbsolutePath)
	}
	return path, nil
}
