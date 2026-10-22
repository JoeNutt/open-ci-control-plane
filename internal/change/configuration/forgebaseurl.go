// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

const httpsScheme = "https"

var (
	errNotHTTPS      = errors.New("the address is not https")
	errNoHost        = errors.New("the address names no host")
	errTrailingSlash = errors.New("the address ends in a slash")
	errQuery         = errors.New("the address carries a query")
	errFragment      = errors.New("the address carries a fragment")
	errCredentials   = errors.New("the address carries credentials")
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
	if fault := originFaultOf(parsed); fault != nil {
		return url.URL{}, forgeBaseURLItem.invalid(fault)
	}
	return *parsed, nil
}

type addressFault struct {
	reason error
	holds  bool
}

func originFaultOf(address *url.URL) error {
	faults := []addressFault{
		{errNotHTTPS, address.Scheme != httpsScheme},
		{errNoHost, address.Host == ""},
		{errTrailingSlash, strings.HasSuffix(address.Path, "/")},
		{errQuery, address.RawQuery != "" || address.ForceQuery},
		{errFragment, address.Fragment != ""},
		{errCredentials, address.User != nil},
	}
	held := slices.IndexFunc(faults, func(fault addressFault) bool { return fault.holds })
	if held < 0 {
		return nil
	}
	return faults[held].reason
}
