package conversionrules

import (
	"strings"
)

const pathSeparator = "."

// matchPath returns true if the generic path representation matches concrete path.
// The generic path can contain special identifier for wildcard array index: '#'.
func matchPath(generic, concrete string) bool {
	if generic == concrete {
		return true
	}
	ys := strings.Split(generic, pathSeparator)
	cs := strings.Split(concrete, pathSeparator)

	if len(ys) != len(cs) {
		return false
	}
	for i := range ys {
		if ys[i] == cs[i] {
			continue
		}
		if ys[i] == "#" {
			continue
		}
		return false
	}
	return true
}
