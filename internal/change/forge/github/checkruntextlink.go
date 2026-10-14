// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"fmt"
	"net/url"
	"strings"
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

func checkRunTextDocumentationOf(stated string) string {
	if !checkRunTextIsPlainHTTPSLink(stated) {
		return fmt.Sprintf(checkRunTextNotALinkLayout, markdownCodeSpanOf(stated))
	}
	return fmt.Sprintf(checkRunTextAutolinkLayout, stated)
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

func checkRunTextIsDNSName(host string) bool {
	if strings.HasPrefix(host, "[") || host == "localhost" {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, character := range label {
			if !(character == '-' || ('0' <= character && character <= '9') ||
				('a' <= character && character <= 'z') || ('A' <= character && character <= 'Z')) {
				return false
			}
		}
	}
	last := labels[len(labels)-1]
	return strings.Trim(last, "0123456789") != ""
}

func checkRunTextHasOnlyLinkCharacters(stated string) bool {
	for _, character := range stated {
		if !strings.ContainsRune(checkRunTextLinkCharacters, character) {
			return false
		}
	}
	return true
}
