package openslotonobl9

import (
	"fmt"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
)

var (
	opensloInlineReferenceConfig = openslosdk.ReferenceConfig{
		V1: openslosdk.ReferenceConfigV1{
			SLO: openslosdk.ReferenceConfigV1SLO{
				SLI: true,
			},
			AlertPolicy: openslosdk.ReferenceConfigV1AlertPolicy{
				AlertCondition: true,
			},
		},
	}
	opensloExportReferenceConfig = openslosdk.ReferenceConfig{
		V1: openslosdk.ReferenceConfigV1{
			SLO: openslosdk.ReferenceConfigV1SLO{
				AlertPolicy: true,
			},
			AlertPolicy: openslosdk.ReferenceConfigV1AlertPolicy{
				AlertNotificationTarget: true,
			},
		},
	}
)

func resolveObjectReferences(objects []openslo.Object) ([]openslo.Object, error) {
	objects, err := openslosdk.NewReferenceInliner(objects...).
		RemoveReferencedObjects().
		WithConfig(opensloInlineReferenceConfig).
		Inline()
	if err != nil {
		return nil, fmt.Errorf("failed to inline OpenSLO referenced objects: %w", err)
	}

	objects = openslosdk.NewReferenceExporter(objects...).
		WithConfig(opensloExportReferenceConfig).
		Export()
	return objects, nil
}
