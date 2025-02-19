package conversionrules

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

type Rules map[string]Converter

func (r Rules) HasRuleFor(key string) bool {
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

	data, err := json.Marshal(v)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal value for path %s", path)
	}
	return sjson.Set(jsonObject, "metadata.annotations."+path, string(data))
}
