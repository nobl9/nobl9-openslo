package conversionrules

type Rules map[string]Converter

func (r Rules) Convert(jsonObject, path string, v any) (string, error) {
	if rule, ok := r[path]; ok {
		return rule.Convert(jsonObject, path, v)
	}
	for rulePath := range r {
		if matchPath(rulePath, path) {
			return r[rulePath].Convert(jsonObject, path, v)
		}
	}
	return jsonObject, nil
}
