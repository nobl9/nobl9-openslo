package openslotonobl9

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/agent"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/alertmethod"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/direct"
	"github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/pkg/errors"
)

var opensloObjectValidation = govy.New(
	govy.For(func(o openslo.Object) openslo.Version { return o.GetVersion() }).
		WithName("apiVersion").
		Rules(rules.OneOf(openslo.VersionV1)),
	govy.For(govy.GetSelf[openslo.Object]()).
		When(func(o openslo.Object) bool { return o.GetVersion() == openslo.VersionV1 }).
		Include(
			opensloV1AnnotationsValidation,
			opensloV1Validation,
		),
).
	WithNameFunc(func(o openslo.Object) string {
		return fmt.Sprintf("%s.%s %s", o.GetVersion(), o.GetKind(), o.GetName())
	})

var opensloV1Validation = govy.New(
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.Service]).
		When(whenObjectIsKind(openslo.KindService)),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.SLO]).
		When(whenObjectIsKind(openslo.KindSLO)).
		Include(opensloV1SLOValidation),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.DataSource]).
		When(whenObjectIsKind(openslo.KindDataSource)).
		Include(opensloV1DataSourceValidation),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.AlertNotificationTarget]).
		When(whenObjectIsKind(openslo.KindAlertNotificationTarget)).
		Include(govy.New(
			govy.For(func(a v1.AlertNotificationTarget) string { return a.Spec.Target }).
				WithName("spec.target").
				Rules(rules.OneOf(slices.Sorted(maps.Keys(getAlertMethodTypes()))...)),
		)),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.AlertPolicy]).
		When(whenObjectIsKind(openslo.KindAlertPolicy)),
)

var opensloV1AnnotationsValidation = govy.New(
	govy.Transform(
		govy.GetSelf[openslo.Object](),
		func(o openslo.Object) (v1.Object, error) { return o.(v1.Object), nil },
	).
		Include(govy.New(
			govy.For(func(o v1.Object) v1.Metadata { return o.GetMetadata() }).
				WithName("metadata").
				Include(govy.New(
					govy.ForMap(func(m v1.Metadata) v1.Annotations { return m.Annotations }).
						WithName("annotations").
						RulesForKeys(rules.NotOneOf(DomainNobl9+"/kind", DomainNobl9+"/apiVersion")),
				)),
		)),
).
	When(func(o openslo.Object) bool { return o.GetKind() != openslo.KindDataSource })

var opensloV1SLOValidation = govy.New(
	govy.ForPointer(func(s v1.SLO) *v1.SLOIndicatorInline { return s.Spec.Indicator }).
		WithName("spec.indicator").
		Include(govy.New(
			govy.For(func(s v1.SLOIndicatorInline) v1.SLISpec { return s.Spec }).
				WithName("spec").
				Include(opensloV1SLIValidation),
		)),
)

var opensloV1SLIValidation = govy.New(
	govy.ForPointer(func(s v1.SLISpec) *v1.SLIMetricSpec { return s.ThresholdMetric }).
		WithName("thresholdMetric").
		Include(opensloV1SLIMetricSpecValidation),
	govy.ForPointer(func(s v1.SLISpec) *v1.SLIRatioMetric { return s.RatioMetric }).
		WithName("ratioMetric").
		Include(govy.New(
			govy.ForPointer(func(s v1.SLIRatioMetric) *v1.SLIMetricSpec { return s.Total }).
				WithName("total").
				Include(opensloV1SLIMetricSpecValidation),
			govy.ForPointer(func(s v1.SLIRatioMetric) *v1.SLIMetricSpec { return s.Good }).
				WithName("good").
				Include(opensloV1SLIMetricSpecValidation),
			govy.ForPointer(func(s v1.SLIRatioMetric) *v1.SLIMetricSpec { return s.Bad }).
				WithName("bad").
				Include(opensloV1SLIMetricSpecValidation),
		)),
)

var opensloV1SLIMetricSpecValidation = govy.New(
	govy.For(func(s v1.SLIMetricSpec) v1.SLIMetricSource { return s.MetricSource }).
		WithName("metricSource").
		Include(govy.New(
			govy.For(func(s v1.SLIMetricSource) string { return s.MetricSourceRef }).
				WithName("metricSourceRef").
				Required().
				Rules(rules.StringNotEmpty()),
			govy.For(func(s v1.SLIMetricSource) string { return s.Type }).
				WithName("type").
				Rules(rules.OneOf(getMetricSpecTypeNames()...)),
		)),
)

var opensloV1DataSourceValidation = govy.New(
	govy.ForMap(func(d v1.DataSource) v1.Annotations { return d.Metadata.Annotations }).
		WithName("spec.annotations").
		RulesForKeys(rules.NEQ(DomainNobl9+"/apiVersion")).
		IncludeForItems(govy.New(
			govy.For(func(m govy.MapItem[string, string]) string { return m.Value }).
				When(func(m govy.MapItem[string, string]) bool { return m.Key == DomainNobl9+"/kind" }).
				Rules(rules.OneOf(manifest.KindAgent.String(), manifest.KindDirect.String())),
		)),
	govy.For(func(d v1.DataSource) string { return d.Spec.Type }).
		When(func(d v1.DataSource) bool {
			return hasAnnotation(d.Metadata.Annotations, DomainNobl9+"/kind", manifest.KindDirect.String())
		}).
		WithName("spec.type").
		Rules(rules.OneOf(getDataSourceTypeNames(manifest.KindDirect)...)),
	govy.For(func(d v1.DataSource) string { return d.Spec.Type }).
		When(func(d v1.DataSource) bool {
			return !hasAnnotation(d.Metadata.Annotations, DomainNobl9+"/kind", manifest.KindDirect.String())
		}).
		WithName("spec.type").
		Rules(rules.OneOf(getDataSourceTypeNames(manifest.KindAgent)...)),
)

func getMetricSpecTypeNames() []string {
	rt := reflect.TypeOf(slo.MetricSpec{})
	names := make([]string, 0, rt.NumField())
	for i := range rt.NumField() {
		tag := rt.Field(i).Tag.Get("json")
		split := strings.Split(tag, ",")
		if len(split) == 0 {
			continue
		}
		fieldName := split[0]
		names = append(names, fieldName)
	}
	return names
}

func getAlertMethodTypes() map[string]any {
	rt := reflect.TypeOf(alertmethod.Spec{})
	types := make(map[string]any, rt.NumField())
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
		typ := field.Type
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		types[fieldName] = reflect.Zero(typ).Interface()
	}
	return types
}

func getDataSourceTypeNames(kind manifest.Kind) []string {
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
		names = append(names, fieldName)
	}
	return names
}

func objectTransformer[T openslo.Object](o openslo.Object) (T, error) {
	v, ok := o.(T)
	if !ok {
		return v, errors.Errorf("failed to cast OpenSLO object to %T", v)
	}
	return v, nil
}

func whenObjectIsKind(kind openslo.Kind) govy.Predicate[openslo.Object] {
	return func(o openslo.Object) bool {
		return o.GetKind() == kind
	}
}

func hasAnnotation(annotations v1.Annotations, key, value string) bool {
	if len(annotations) == 0 {
		return false
	}
	if val, ok := annotations[key]; ok {
		return val == value
	}
	return false
}
