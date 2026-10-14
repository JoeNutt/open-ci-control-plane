// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "errors"

var ErrMalformedHeadSHA = errors.New("the head commit is not a parsed commit SHA")
