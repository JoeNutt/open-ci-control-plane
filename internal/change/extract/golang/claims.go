// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import "strings"

type Extractor struct{}

func (Extractor) Claims(path string) bool {
	return strings.HasSuffix(path, ".go")
}
