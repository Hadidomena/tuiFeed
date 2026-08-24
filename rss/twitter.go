package rss

import (
	"fmt"
	"regexp"
	"strings"
)

const DefaultNitterBase = "https://xcancel.com"

var (
	twitterLinkRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?(?:x|twitter)\.com/@?([A-Za-z0-9_]{1,15})(?:/(with_replies|media|highlights))?/?$`)
	twitterTabRe  = regexp.MustCompile(`(?i)^(with_replies|media|highlights)$`)
	twitterUserRe = regexp.MustCompile(`(?i)^@?([A-Za-z0-9_]{1,15})(?:/(with_replies|media|highlights))?$`)
)

func NormalizeTwitterInput(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	if m := twitterLinkRe.FindStringSubmatch(input); m != nil {
		return nitterRSSURL(m[1], m[2])
	}
	if m := twitterUserRe.FindStringSubmatch(input); m != nil {
		return nitterRSSURL(m[1], m[2])
	}
	return input
}

func nitterRSSURL(user, tab string) string {
	user = strings.TrimPrefix(user, "@")
	tab = strings.ToLower(tab)
	if !twitterTabRe.MatchString(tab) {
		tab = ""
	} else {
		tab = "/" + tab
	}
	return fmt.Sprintf("%s/%s%s/rss", DefaultNitterBase, user, tab)
}
