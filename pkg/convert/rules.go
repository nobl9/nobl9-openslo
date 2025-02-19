package convert

import (
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/twindow"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"

	"github.com/nobl9/nobl9-openslo/internal/conversionrules"
)

func getConversionRules(version, kind string) (conversionrules.Rules, error) {
	switch version {
	case openslo.VersionV1.String():
		switch kind {
		case openslo.KindSLO.String():
			return commonRules, nil
		case openslo.KindService.String():
			return mergeConversionRules(commonRules, sloV1Rules), nil
		default:
			return nil, errors.Errorf("unsupported kind %s for version %s", kind, version)
		}
	default:
		return nil, errors.Errorf("unsupported API version %s", version)
	}
}

var commonRules = conversionrules.Rules{
	"apiVersion": conversionrules.Value(
		func(v any) (any, error) { return manifest.VersionV1alpha.String(), nil },
	),
	"kind":                 conversionrules.Noop(),
	"metadata.name":        conversionrules.Noop(),
	"metadata.displayName": conversionrules.Noop(),
	"metadata.labels":      conversionrules.Noop(),
	"metadata.annotations": conversionrules.Full(convertAnnotations),
	"spec.description":     conversionrules.Noop(),
}

var sloV1Rules = conversionrules.Rules{
	"spec.service":                            conversionrules.Noop(),
	"spec.budgetingMethod":                    conversionrules.Noop(),
	"spec.indicator.spec.ratioMetric.counter": conversionrules.Path("spec.objectives.#.countMetrics.incremental"),
	"spec.indicator.spec.ratioMetric.good":    conversionrules.Path("spec.objectives.#.countMetrics.good"),
	"spec.indicator.spec.ratioMetric.total":   conversionrules.Path("spec.objectives.#.countMetrics.total"),
	"spec.objectives.#.displayName":           conversionrules.Noop(),
	"spec.objectives.#.timeSliceTarget":       conversionrules.Noop(),
	"spec.objectives.#.target":                conversionrules.Noop(),
	"spec.timeWindow.#.duration":              conversionrules.Full(convertSLOTimeWindowDuration),
	"spec.timeWindow.#.isRolling":             conversionrules.Path("spec.timeWindows.#.isRolling"),
	"spec.timeWindow.#.calendar":              conversionrules.Path("spec.timeWindows.#.calendar"),
}

const nobl9AnnotationPrefix = "nobl9.com/"

func convertAnnotations(jsonObject, path string, v any) (updatedJSON string, err error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", errors.Errorf("invalid type for %s, expected map[string]any, got %T", path, v)
	}
	for key, av := range m {
		if strings.HasPrefix(key, nobl9AnnotationPrefix) {
			nobl9ObjectPath := key[len(nobl9AnnotationPrefix):]
			jsonObject, err = sjson.Set(jsonObject, nobl9ObjectPath, av)
		} else {
			// Escape dots in the key to avoid interpreting them as a path.
			key = strings.ReplaceAll(key, ".", "\\.")
			jsonObject, err = sjson.Set(jsonObject, path+"."+key, av)
		}
	}
	return jsonObject, nil
}

func convertSLOTimeWindowDuration(jsonObject, path string, v any) (updatedJSON string, err error) {
	duration, ok := v.(string)
	if !ok {
		return "", errors.Errorf("invalid type for %s, expected string, got %T", path, v)
	}
	parsedDuration, err := v1.ParseDurationShorthand(duration)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s as %T", duration, parsedDuration)
	}
	unit, value := durationShorthandUnitToTimeWindowUnit(parsedDuration)
	jsonObject, err = sjson.Set(jsonObject, "spec.timeWindows.#.unit", unit)
	if err != nil {
		return "", err
	}
	return sjson.Set(jsonObject, "spec.timeWindows.#.value", value)
}

func durationShorthandUnitToTimeWindowUnit(duration v1.DurationShorthand) (string, int) {
	switch duration.GetUnit() {
	case v1.DurationShorthandUnitMinute:
		return twindow.Minute.String(), duration.GetValue()
	case v1.DurationShorthandUnitHour:
		return twindow.Hour.String(), duration.GetValue()
	case v1.DurationShorthandUnitDay:
		return twindow.Day.String(), duration.GetValue()
	case v1.DurationShorthandUnitWeek:
		return twindow.Week.String(), duration.GetValue()
	case v1.DurationShorthandUnitMonth:
		return twindow.Month.String(), duration.GetValue()
	case v1.DurationShorthandUnitQuarter:
		return twindow.Quarter.String(), duration.GetValue()
	case v1.DurationShorthandUnitYear:
		return twindow.Year.String(), duration.GetValue()
	default:
		return "", 0
	}
}

func mergeConversionRules(rules ...conversionrules.Rules) conversionrules.Rules {
	merged := make(conversionrules.Rules)
	for _, r := range rules {
		for k, v := range r {
			merged[k] = v
		}
	}
	return merged
}
