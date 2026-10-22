// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

func forgeBaseURLIn(stated statedItems) (url.URL, error) {
	text, err := valueOf[string](stated, forgeBaseURLItem)
	if err != nil {
		return url.URL{}, err
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return url.URL{}, forgeBaseURLItem.invalid(err)
	}
	faults := []struct {
		reason string
		holds  bool
	}{
		{"is not https", parsed.Scheme != "https"},
		{"names no host", parsed.Host == ""},
		{"ends in a slash", strings.HasSuffix(parsed.Path, "/")},
		{"carries a query", parsed.RawQuery != "" || parsed.ForceQuery},
		{"carries a fragment", parsed.Fragment != ""},
		{"carries credentials", parsed.User != nil},
	}
	held := slices.IndexFunc(faults, func(fault struct {
		reason string
		holds  bool
	}) bool {
		return fault.holds
	})
	if held >= 0 {
		return url.URL{}, forgeBaseURLItem.invalid(errors.New(faults[held].reason))
	}
	return *parsed, nil
}
