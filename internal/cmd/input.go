package cmd

import "regexp"

type LoadInput struct {
	ExcludePatterns []*regexp.Regexp
}

// Matches reports whether path matches any of the exclude patterns.
func (inp *LoadInput) Matches(path string) bool {
	for _, re := range inp.ExcludePatterns {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
