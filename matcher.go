package rsync

import (
	"regexp"
)

type matcher struct {
	regExp *regexp.Regexp
}

func (m matcher) Match(data string) bool {
	return m.regExp.Match([]byte(data))
}

func (m matcher) Extract(data string, pos int) string {
	const submatchCount = 1
	matches := m.regExp.FindAllStringSubmatch(data, submatchCount)
	if len(matches) == 0 || len(matches[0]) < pos+1 {
		return ""
	}

	return matches[0][pos]
}

func newMatcher(regExpString string) *matcher {
	return &matcher{
		regExp: regexp.MustCompile(regExpString),
	}
}
