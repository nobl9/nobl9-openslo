package openslotonobl9

import (
	"slices"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
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
