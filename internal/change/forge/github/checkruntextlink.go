// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"net/url"
	"strings"
)

// RFC 1035 section 2.3.1: a label is letters, digits and hyphens, starts and ends
// with a letter or digit, and is at most 63 characters.
const (
	checkRunTextMaxLabelLength  = 63
	checkRunTextMinimumLabels   = 2
	checkRunTextDigits          = "0123456789"
	checkRunTextLabelCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" + checkRunTextDigits + "-"
)

const (
	checkRunTextLinkPrefix      = "https://"
	checkRunTextLinkDefaultPort = "443"
	checkRunTextLinkMaxLength   = 2048
	checkRunTextNotALinkLayout  = "%s (not a link)"
	checkRunTextAutolinkLayout  = "<%s>"
)

// RFC 3986 section 2.3 unreserved characters, plus the delimiters a plain
// documentation URL needs. CommonMark 0.31.2 section 6.5: an autolink may not
// contain a space, '<', '>' or an ASCII control character; none is in this set.
const checkRunTextLinkCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789" +
	"-._~" + "/?=#:%"

type checkRunTextLink struct {
	address string
}

func checkRunTextLinkOf(stated string) (checkRunTextLink, bool) {
	if !checkRunTextIsPlainHTTPSLink(stated) {
		return checkRunTextLink{}, false
	}
	return checkRunTextLink{address: stated}, true
}

func checkRunTextDocumentationOf(stated string) string {
	link, isLink := checkRunTextLinkOf(stated)
	if !isLink {
		return fmt.Sprintf(checkRunTextNotALinkLayout, markdownCodeSpanOf(stated))
	}
	return fmt.Sprintf(checkRunTextAutolinkLayout, link.address)
}

func checkRunTextIsPlainHTTPSLink(stated string) bool {
	if len(stated) > checkRunTextLinkMaxLength || !strings.HasPrefix(stated, checkRunTextLinkPrefix) {
		return false
	}
	if !checkRunTextHasOnlyLinkCharacters(stated) {
		return false
	}
	parsed, err := url.Parse(stated)
	if err != nil {
		return false
	}
	return parsed.Host != "" && parsed.User == nil &&
		(parsed.Port() == "" || parsed.Port() == checkRunTextLinkDefaultPort) &&
		checkRunTextIsDNSName(parsed.Hostname())
}

func checkRunTextHasOnlyLinkCharacters(stated string) bool {
	for _, character := range stated {
		if !strings.ContainsRune(checkRunTextLinkCharacters, character) {
			return false
		}
	}
	return true
}

func checkRunTextIsDNSName(host string) bool {
	labels := strings.Split(host, ".")
	if len(labels) < checkRunTextMinimumLabels || !checkRunTextAreDNSLabels(labels) {
		return false
	}
	return strings.Trim(labels[len(labels)-1], checkRunTextDigits) != ""
}

func checkRunTextAreDNSLabels(labels []string) bool {
	for _, label := range labels {
		if !checkRunTextIsDNSLabel(label) {
			return false
		}
	}
	return true
}

func checkRunTextIsDNSLabel(label string) bool {
	if label == "" || len(label) > checkRunTextMaxLabelLength {
		return false
	}
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}
	return strings.Trim(label, checkRunTextLabelCharacters) == ""
}
