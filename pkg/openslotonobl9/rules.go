package openslotonobl9

import (
	"encoding/json"
	"maps"
	"reflect"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/agent"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/alertmethod"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/alertpolicy"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/direct"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/twindow"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"

	"github.com/nobl9/nobl9-openslo/internal/conversionrules"
	"github.com/nobl9/nobl9-openslo/internal/jsonpath"
)

func getConversionRules(version openslo.Version, kind openslo.Kind) (conversionrules.Rules, error) {
	switch version {
	case openslo.VersionV1:
		switch kind {
		case openslo.KindSLO:
			return mergeConversionRules(v1CommonRules, v1SLORules), nil
		case openslo.KindService:
			return v1CommonRules, nil
		case openslo.KindDataSource:
			return mergeConversionRules(v1CommonRules, v1DataSourceRules), nil
		case openslo.KindAlertPolicy:
			return mergeConversionRules(v1CommonRules, v1AlertPolicyRules), nil
		case openslo.KindAlertNotificationTarget:
			return mergeConversionRules(v1CommonRules, v1AlertNotificationTargetRules), nil
		case openslo.KindSLI:
			return nil, nil
		case openslo.KindAlertCondition:
			return nil, nil
		default:
			return nil, errors.Errorf("unsupported kind %s for version %s", kind, version)
		}
	default:
		return nil, errors.Errorf("unsupported API version %s", version)
	}
}

var v1CommonRules = conversionrules.Rules{
	"apiVersion": conversionrules.Value(func(v any) (any, error) {
		return manifest.VersionV1alpha.String(), nil
	}),
	"kind":                 conversionrules.Direct(),
	"metadata.name":        conversionrules.Direct(),
	"metadata.displayName": conversionrules.Direct(),
	"metadata.labels":      conversionrules.Direct(),
	"metadata.annotations": conversionrules.Custom(convertAnnotations),
	"spec.description":     conversionrules.Direct(),
}

var v1SLORules = conversionrules.Rules{
	"spec.service":                                       conversionrules.Direct(),
	"spec.budgetingMethod":                               conversionrules.Direct(),
	"spec.indicator.metadata.name":                       conversionrules.Annotation(),
	"spec.indicator.metadata.displayName":                conversionrules.Annotation(),
	"spec.indicator.spec.ratioMetric.counter":            conversionrules.Path("spec.objectives.#.countMetrics.incremental"),
	"spec.indicator.spec.ratioMetric.total.metricSource": conversionrules.Custom(convertSLOMetricSource(sliMetricTypeTotal)),
	"spec.indicator.spec.ratioMetric.good.metricSource":  conversionrules.Custom(convertSLOMetricSource(sliMetricTypeGood)),
	"spec.indicator.spec.ratioMetric.bad.metricSource":   conversionrules.Custom(convertSLOMetricSource(sliMetricTypeBad)),
	"spec.indicator.spec.thresholdMetric.metricSource":   conversionrules.Custom(convertSLOMetricSource(sliMetricTypeRaw)),
	"spec.objectives.#.displayName":                      conversionrules.Direct(),
	"spec.objectives.#.timeSliceTarget":                  conversionrules.Direct(),
	"spec.objectives.#.timeSliceWindow":                  conversionrules.Direct(),
	"spec.objectives.#.target":                           conversionrules.Direct(),
	"spec.objectives.#.op":                               conversionrules.Direct(),
	"spec.timeWindow.0.duration":                         conversionrules.Custom(convertSLOTimeWindowDuration),
	"spec.timeWindow.0.isRolling":                        conversionrules.Path("spec.timeWindows.0.isRolling"),
	"spec.timeWindow.0.calendar":                         conversionrules.Path("spec.timeWindows.0.calendar"),
}

var v1DataSourceRules = conversionrules.Rules{
	"kind": conversionrules.Value(func(any) (any, error) {
		return manifest.KindAgent.String(), nil
	}),
	"spec": conversionrules.Custom(convertDataSourceSpec),
}

// TODO:
// - Ensure only one severity is set per whole alert policy.
var v1AlertPolicyRules = conversionrules.Rules{
	"spec.conditions.#.spec.severity":                 conversionrules.Path("spec.severity"),
	"spec.conditions.#.spec.condition.op":             conversionrules.PathIndex("spec.conditions.%d.op"),
	"spec.conditions.#.spec.condition.kind":           conversionrules.Custom(convertConditionKind),
	"spec.conditions.#.spec.condition.threshold":      conversionrules.PathIndex("spec.conditions.%d.value"),
	"spec.conditions.#.spec.condition.lookbackWindow": conversionrules.PathIndex("spec.conditions.%d.alertingWindow"),
	"spec.conditions.#.spec.condition.alertAfter":     conversionrules.PathIndex("spec.conditions.%d.lastsFor"),
	"spec.notificationTargets.#.targetRef":            conversionrules.PathIndex("spec.alertMethods.%d.metadata.name"),
}

var v1AlertNotificationTargetRules = conversionrules.Rules{
	"kind": conversionrules.Value(func(any) (any, error) {
		return manifest.KindAlertMethod.String(), nil
	}),
	"spec.target": conversionrules.Custom(convertNotificationTarget),
}

const nobl9AnnotationPrefix = "nobl9.com/"

func convertAnnotations(jsonObject, path string, v any) (updatedJSON string, err error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", errors.Errorf("invalid type for %s, expected map[string]any, got %T", path, v)
	}
	for key, av := range m {
		var newPath string
		switch {
		case strings.HasPrefix(key, nobl9AnnotationPrefix):
			newPath = key[len(nobl9AnnotationPrefix):]
		default:
			// Escape dots in the key to avoid interpreting them as a path.
			key = strings.ReplaceAll(key, ".", "\\.")
			newPath = path + "." + key
		}
		jsonObject, err = sjson.Set(jsonObject, newPath, av)
		if err != nil {
			return "", err
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
	jsonObject, err = sjson.Set(jsonObject, "spec.timeWindows.0.unit", unit)
	if err != nil {
		return "", err
	}
	return sjson.Set(jsonObject, "spec.timeWindows.0.count", value)
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

type sliMetricType int

const (
	sliMetricTypeRaw sliMetricType = iota + 1
	sliMetricTypeTotal
	sliMetricTypeGood
	sliMetricTypeBad
)

func convertSLOMetricSource(typ sliMetricType) conversionrules.ConversionFunc {
	return func(jsonObject, path string, v any) (updatedJSON string, err error) {
		metricSource, err := anyToType[v1.SLIMetricSource](v)
		if err != nil {
			return "", err
		}
		if err = validateMetricSpecName(metricSource.Type); err != nil {
			return "", err
		}
		var newPath string
		switch typ {
		case sliMetricTypeRaw:
			newPath = "spec.objectives.#.rawMetric.query"
		case sliMetricTypeTotal:
			newPath = "spec.objectives.#.countMetrics.total"
		case sliMetricTypeGood:
			newPath = "spec.objectives.#.countMetrics.good"
		case sliMetricTypeBad:
			newPath = "spec.objectives.#.countMetrics.bad"
		default:
			return "", errors.Errorf("unsupported metric source type %d", typ)
		}
		newPath += "." + metricSource.Type
		jsonObject, err = jsonpath.Set(jsonObject, newPath, metricSource.Spec)
		if err != nil {
			return "", err
		}
		jsonObject, err = jsonpath.Set(jsonObject, "spec.indicator.metricSource.name", metricSource.MetricSourceRef)
		if err != nil {
			return "", err
		}
		return jsonObject, nil
	}
}

func validateMetricSpecName(name string) error {
	rt := reflect.TypeOf(slo.MetricSpec{})
	names := make([]string, 0, rt.NumField())
	for i := range rt.NumField() {
		tag := rt.Field(i).Tag.Get("json")
		split := strings.Split(tag, ",")
		if len(split) == 0 {
			continue
		}
		fieldName := split[0]
		if fieldName == name {
			return nil
		}
		names = append(names, fieldName)
	}
	return errors.Errorf("unsupported metric spec name %s, try one of: %s", name, strings.Join(names, ", "))
}

func convertDataSourceSpec(jsonObject, _ string, v any) (updatedJSON string, err error) {
	spec, err := anyToType[v1.DataSourceSpec](v)
	if err != nil {
		return "", err
	}
	if err = validateDataSourceTypeName(manifest.KindAgent, spec.Type); err != nil {
		return "", err
	}
	return sjson.SetRaw(jsonObject, "spec."+spec.Type, string(spec.ConnectionDetails))
}

func convertConditionKind(jsonObject, path string, v any) (updatedJSON string, err error) {
	kind, ok := v.(string)
	if !ok {
		return "", errors.Errorf("invalid type for %s, expected string, got %T", path, v)
	}
	if kind != "burnrate" {
		return "", errors.Errorf("unsupported condition kind '%s', only 'burnrate' is supported", kind)
	}
	newPath := strings.TrimSuffix(path, "spec.condition.kind") + "measurement"
	return sjson.Set(jsonObject, newPath, alertpolicy.MeasurementAverageBurnRate.String())
}

func convertNotificationTarget(jsonObject, path string, v any) (updatedJSON string, err error) {
	target, ok := v.(string)
	if !ok {
		return "", errors.Errorf("invalid type for %s, expected string, got %T", path, v)
	}
	alertMethod, err := getAlertMethodTypeForName(target)
	if err != nil {
		return "", err
	}
	return sjson.Set(jsonObject, "spec."+target, alertMethod)
}

func validateDataSourceTypeName(kind manifest.Kind, name string) error {
	var rt reflect.Type
	switch kind {
	case manifest.KindDirect:
		rt = reflect.TypeOf(direct.Spec{})
	default:
		rt = reflect.TypeOf(agent.Spec{})
	}
	names := make([]string, 0, rt.NumField())
	for i := range rt.NumField() {
		field := rt.Field(i)
		if !strings.HasSuffix(field.Type.String(), "Config") {
			continue
		}
		tag := field.Tag.Get("json")
		split := strings.Split(tag, ",")
		if len(split) == 0 {
			continue
		}
		fieldName := split[0]
		if fieldName == name {
			return nil
		}
		names = append(names, fieldName)
	}
	return errors.Errorf("unsupported data source type name %s, try one of: %s", name, strings.Join(names, ", "))
}

func getAlertMethodTypeForName(name string) (any, error) {
	rt := reflect.TypeOf(alertmethod.Spec{})
	names := make([]string, 0, rt.NumField())
	for i := range rt.NumField() {
		field := rt.Field(i)
		if !strings.HasSuffix(field.Type.String(), "Method") {
			continue
		}
		tag := field.Tag.Get("json")
		split := strings.Split(tag, ",")
		if len(split) == 0 {
			continue
		}
		fieldName := split[0]
		if fieldName == name {
			typ := field.Type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			return reflect.Zero(typ).Interface(), nil
		}
		names = append(names, fieldName)
	}
	return nil, errors.Errorf("unsupported alert method type name %s, try one of: %s", name, strings.Join(names, ", "))
}

func mergeConversionRules(rules ...conversionrules.Rules) conversionrules.Rules {
	merged := make(conversionrules.Rules)
	for _, r := range rules {
		maps.Copy(merged, r)
	}
	return merged
}

// anyToType converts any value to a specific type.
// It uses JSON conversion as an intermediary step.
func anyToType[T any](v any) (result T, err error) {
	rawJSON, err := json.Marshal(v)
	if err != nil {
		return result, errors.Wrapf(err, "failed to convert %T to %T", v, result)
	}
	if err = json.Unmarshal(rawJSON, &result); err != nil {
		return result, errors.Wrapf(err, "failed to convert %T to %T", v, result)
	}
	return result, nil
}
