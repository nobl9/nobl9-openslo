package conversionrules

import (
	"strconv"
	"strings"

	"github.com/nobl9/nobl9-openslo/internal/annotations"
)

type Rules map[string]Converter

func (r Rules) HasRuleFor(key string) bool {
	key = generifyPath(key)
	if _, ok := r[key]; ok {
		return true
	}
	for k := range r {
		// The field is part of another, parent field for which a rule is defined.
		if strings.HasPrefix(key, k) {
			return false
		}
		// The field is a parent for a field which has a defined rule.
		if strings.HasPrefix(k, key) {
			return false
		}
	}
	// Default rule is to convert to annotation.
	return true
}

func (r Rules) Convert(jsonObject, path string, v any) (string, error) {
	if rule, ok := r[path]; ok {
		return rule.Convert(jsonObject, path, v)
	}
	for rulePath := range r {
		if matchPath(rulePath, path) {
			return r[rulePath].Convert(jsonObject, path, v)
		}
	}
	return annotations.AddOpenSLOToNobl9(jsonObject, path, v)
}

func generifyPath(path string) string {
	if strings.Index(path, pathSeparator) == -1 {
		return path
	}
	split := strings.Split(path, pathSeparator)
	newSlice := make([]string, 0, len(split))
	for _, s := range split {
		if _, err := strconv.Atoi(s); err == nil {
			newSlice = append(newSlice, "#")
			continue
		}
		newSlice = append(newSlice, s)
	}
	return strings.Join(newSlice, pathSeparator)
}
