package openslotonobl9

import (
	"fmt"
	"slices"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

var opensloObjectValidation = govy.New(
	govy.For(func(o openslo.Object) openslo.Version { return o.GetVersion() }).
		WithName("apiVersion").
		Rules(rules.OneOf(openslo.VersionV1)),
	govy.For(govy.GetSelf[openslo.Object]()).
		When(func(o openslo.Object) bool { return o.GetVersion() == openslo.VersionV1 }).
		Include(opensloObjectV1Validation),
).
	WithNameFunc(func(o openslo.Object) string {
		return fmt.Sprintf("%s.%s %s", o.GetVersion(), o.GetKind(), o.GetName())
	})

var opensloObjectV1Validation = govy.New(
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.Service]).
		When(whenObjectIsKind(openslo.KindService)),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.SLO]).
		When(whenObjectIsKind(openslo.KindSLO)),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.DataSource]).
		When(whenObjectIsKind(openslo.KindDataSource)),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.AlertNotificationTarget]).
		When(whenObjectIsKind(openslo.KindAlertNotificationTarget)),
	govy.Transform(govy.GetSelf[openslo.Object](), objectTransformer[v1.AlertPolicy]).
		When(whenObjectIsKind(openslo.KindAlertPolicy)),
)

func validateOpenSLOObject(object gjson.Result, version openslo.Version, kind openslo.Kind) error {
	switch version {
	case openslo.VersionV1:
		return validateOpenSLOV1Object(object, kind)
	}
	return nil
}

var nobl9DataSourceTypes = []string{
	manifest.KindAgent.String(),
	manifest.KindDirect.String(),
}

func validateOpenSLOV1Object(object gjson.Result, kind openslo.Kind) error {
	switch kind {
	case openslo.KindDataSource:
		kindAnnotation := object.Get("metadata.annotations.nobl9.com/kind")
		if kindAnnotation.Exists() && !slices.Contains(nobl9DataSourceTypes, kindAnnotation.String()) {
			return errors.Errorf("nobl9.com/kind must be one of: %s", strings.Join(nobl9DataSourceTypes, ", "))
		}
	case openslo.KindAlertPolicy:

	default:
		if object.Get("metadata.annotations.nobl9.com/kind").Exists() {
			return errors.New("nobl9.com/kind annotation is only allowed for DataSource kind")
		}
	}
	return nil
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
