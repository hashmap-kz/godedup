package cmd

import "regexp"

type LoadInput struct {
	ExcludeRegex *regexp.Regexp
}
