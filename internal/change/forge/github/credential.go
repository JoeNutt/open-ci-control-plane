// SPDX-License-Identifier: GPL-3.0-or-later

package github

type Credential string

func (credential Credential) String() string { return "[redacted]" }

func (credential Credential) GoString() string { return `"[redacted]"` }
