// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "net/url"

func forgeBaseURLIn(stated statedItems) (url.URL, error) {
	text, err := valueOf[string](stated, forgeBaseURLItem)
	if err != nil {
		return url.URL{}, err
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return url.URL{}, forgeBaseURLItem.invalid(err)
	}
	return *parsed, nil
}
