package openslotonobl9

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	v1 "github.com/OpenSLO/go-sdk/pkg/openslo/v1"
	"github.com/OpenSLO/go-sdk/pkg/openslo/v1alpha"
	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/goccy/go-yaml"
	"github.com/nobl9/govy/pkg/govytest"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	inputsDir  = "./test_data/inputs/"
	outputsDir = "./test_data/outputs/"
)

func TestConvert(t *testing.T) {
	inputs := listAllFilesInDir(t, inputsDir)
	outputs := listAllFilesInDir(t, outputsDir)
	require.Len(t, inputs, len(outputs))

	for _, fileName := range inputs {
		t.Run(fileName, func(t *testing.T) {
			inputFileData, err := os.ReadFile(filepath.Join(inputsDir, fileName))
			require.NoError(t, err)

			outputsFileData, err := os.ReadFile(filepath.Join(outputsDir, fileName))
			require.NoError(t, err)
			expectedObjects, err := sdk.DecodeObjects(outputsFileData)
			require.NoError(t, err)
			errs := manifest.Validate(expectedObjects)
			require.Empty(t, errs, "failed to validate Nobl9 objects")

			opensloObjects, err := openslosdk.Decode(bytes.NewReader(inputFileData), openslosdk.FormatYAML)
			require.NoError(t, err)

			actual, err := Convert(opensloObjects)
			require.NoError(t, err)
			var buf bytes.Buffer
			err = sdk.EncodeObjects(actual, &buf, manifest.ObjectFormatJSON)
			require.NoError(t, err)

			expectedJSON, err := yaml.YAMLToJSON(outputsFileData)
			require.NoError(t, err)
			assert.JSONEq(t, string(expectedJSON), buf.String())

			nobl9Objects, err := sdk.DecodeObjects(buf.Bytes())
			require.NoError(t, err)
			errs = manifest.Validate(nobl9Objects)
			require.Empty(t, errs, "failed to validate Nobl9 objects")
		})
	}
}

// Golden path should be covered with [TestConvert].
func TestConvert_Validate(t *testing.T) {
	tests := map[string]struct {
		objects []openslo.Object
		errors  []govytest.ExpectedRuleError
	}{
		"invalid version": {
			objects: []openslo.Object{v1alpha.NewService(
				v1alpha.Metadata{Name: "test"},
				v1alpha.ServiceSpec{},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName: "apiVersion",
					Code:         rules.ErrorCodeOneOf,
				},
			},
		},
		"invalid type for v1.AlertNotificationTarget": {
			objects: []openslo.Object{v1.NewAlertNotificationTarget(
				v1.Metadata{Name: "test"},
				v1.AlertNotificationTargetSpec{
					Target: "foo",
				},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName: "spec.target",
					Code:         rules.ErrorCodeOneOf,
					Message:      "must be one of: discord, email, jira, msteams, opsgenie, pagerduty, servicenow, slack, webhook",
				},
			},
		},
		"valid type for v1.AlertNotificationTarget": {
			objects: []openslo.Object{v1.NewAlertNotificationTarget(
				v1.Metadata{
					Name: "test",
					Annotations: v1.Annotations{
						DomainNobl9 + "/spec.jira.url":        "https://jira.example.com",
						DomainNobl9 + "/spec.jira.username":   "my-user",
						DomainNobl9 + "/spec.jira.projectKey": "secret",
					},
				},
				v1.AlertNotificationTargetSpec{
					Target: "jira",
				},
			)},
		},
		"invalid type for v1.DataSource (default Agent)": {
			objects: []openslo.Object{v1.NewDataSource(
				v1.Metadata{Name: "test"},
				v1.DataSourceSpec{
					Type:              "foo",
					ConnectionDetails: json.RawMessage(`{}`),
				},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName:    "spec.type",
					Code:            rules.ErrorCodeOneOf,
					ContainsMessage: "must be one of: prometheus, datadog",
				},
			},
		},
		"invalid type for v1.DataSource (Agent via annotation)": {
			objects: []openslo.Object{v1.NewDataSource(
				v1.Metadata{
					Name:        "test",
					Annotations: v1.Annotations{DomainNobl9 + "/kind": "Agent"},
				},
				v1.DataSourceSpec{
					Type:              "foo",
					ConnectionDetails: json.RawMessage(`{}`),
				},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName:    "spec.type",
					Code:            rules.ErrorCodeOneOf,
					ContainsMessage: "must be one of: prometheus, datadog",
				},
			},
		},
		"invalid type for v1.DataSource (Direct via annotation)": {
			objects: []openslo.Object{v1.NewDataSource(
				v1.Metadata{
					Name:        "test",
					Annotations: v1.Annotations{DomainNobl9 + "/kind": "Direct"},
				},
				v1.DataSourceSpec{
					Type:              "foo",
					ConnectionDetails: json.RawMessage(`{}`),
				},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName:    "spec.type",
					Code:            rules.ErrorCodeOneOf,
					ContainsMessage: "must be one of: datadog, newRelic",
				},
			},
		},
		"valid type for v1.DataSource (default Agent)": {
			objects: []openslo.Object{v1.NewDataSource(
				v1.Metadata{Name: "test"},
				v1.DataSourceSpec{
					Type:              "prometheus",
					ConnectionDetails: json.RawMessage(`{"url": "https://example.com"}`),
				},
			)},
		},
		"valid type for v1.DataSource (Direct via annotation)": {
			objects: []openslo.Object{v1.NewDataSource(
				v1.Metadata{
					Name:        "test",
					Annotations: v1.Annotations{DomainNobl9 + "/kind": "Direct"},
				},
				v1.DataSourceSpec{
					Type:              "datadog",
					ConnectionDetails: json.RawMessage(`{"site": "eu"}`),
				},
			)},
		},
		"forbidden kind annotation": {
			objects: []openslo.Object{v1.NewService(
				v1.Metadata{
					Name:        "test",
					Annotations: v1.Annotations{DomainNobl9 + "/kind": "MyType"},
				},
				v1.ServiceSpec{},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName: "metadata.annotations.['nobl9.com/kind']",
					IsKeyError:   true,
					Code:         rules.ErrorCodeNotOneOf,
				},
			},
		},
		"forbidden apiVersion annotation": {
			objects: []openslo.Object{v1.NewService(
				v1.Metadata{
					Name:        "test",
					Annotations: v1.Annotations{DomainNobl9 + "/apiVersion": "myVersion"},
				},
				v1.ServiceSpec{},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName: "metadata.annotations.['nobl9.com/apiVersion']",
					IsKeyError:   true,
					Code:         rules.ErrorCodeNotOneOf,
				},
			},
		},
		"invalid type and missing metricSourceRef for v1.SLO": {
			objects: []openslo.Object{v1.NewSLO(
				v1.Metadata{Name: "test"},
				v1.SLOSpec{
					Service: "web",
					Indicator: &v1.SLOIndicatorInline{
						Metadata: v1.Metadata{
							Name: "web-successful-requests-ratio",
						},
						Spec: v1.SLISpec{
							RatioMetric: &v1.SLIRatioMetric{
								Counter: true,
								Good: &v1.SLIMetricSpec{
									MetricSource: v1.SLIMetricSource{
										Type: "Prometheus",
										Spec: map[string]any{
											"query": `sum(http_requests{k8s_cluster="prod",component="web",code=~"2xx|4xx"})`,
										},
									},
								},
								Total: &v1.SLIMetricSpec{
									MetricSource: v1.SLIMetricSource{
										Type: "Prometheus",
										Spec: map[string]any{
											"query": `sum(http_requests{k8s_cluster="prod",component="web"})`,
										},
									},
								},
							},
						},
					},
					TimeWindow: []v1.SLOTimeWindow{
						{
							Duration:  v1.NewDurationShorthand(1, v1.DurationShorthandUnitWeek),
							IsRolling: false,
							Calendar: &v1.SLOCalendar{
								StartTime: "2022-01-01 12:00:00",
								TimeZone:  "America/New_York",
							},
						},
					},
					BudgetingMethod: v1.SLOBudgetingMethodTimeslices,
					Objectives: []v1.SLOObjective{
						{
							DisplayName:     "Good",
							Target:          ptr(0.995),
							TimeSliceTarget: ptr(0.95),
							TimeSliceWindow: ptr(v1.NewDurationShorthand(1, v1.DurationShorthandUnitMinute)),
						},
					},
				},
			)},
			errors: []govytest.ExpectedRuleError{
				{
					PropertyName:    "spec.indicator.spec.ratioMetric.total.metricSource.type",
					Code:            rules.ErrorCodeOneOf,
					ContainsMessage: "must be one of: prometheus, datadog",
				},
				{
					PropertyName:    "spec.indicator.spec.ratioMetric.good.metricSource.type",
					Code:            rules.ErrorCodeOneOf,
					ContainsMessage: "must be one of: prometheus, datadog",
				},
				{
					PropertyName: "spec.indicator.spec.ratioMetric.total.metricSource.metricSourceRef",
					Code:         rules.ErrorCodeRequired,
				},
				{
					PropertyName: "spec.indicator.spec.ratioMetric.good.metricSource.metricSourceRef",
					Code:         rules.ErrorCodeRequired,
				},
			},
		},
		"valid type for v1.SLO": {
			objects: []openslo.Object{v1.NewSLO(
				v1.Metadata{Name: "test"},
				v1.SLOSpec{
					Service: "web",
					Indicator: &v1.SLOIndicatorInline{
						Metadata: v1.Metadata{
							Name: "web-successful-requests-ratio",
						},
						Spec: v1.SLISpec{
							RatioMetric: &v1.SLIRatioMetric{
								Counter: true,
								Good: &v1.SLIMetricSpec{
									MetricSource: v1.SLIMetricSource{
										Type:            "prometheus",
										MetricSourceRef: "foo",
										Spec: map[string]any{
											"promql": `sum(http_requests{k8s_cluster="prod",component="web",code=~"2xx|4xx"})`,
										},
									},
								},
								Total: &v1.SLIMetricSpec{
									MetricSource: v1.SLIMetricSource{
										Type:            "prometheus",
										MetricSourceRef: "foo",
										Spec: map[string]any{
											"promql": `sum(http_requests{k8s_cluster="prod",component="web"})`,
										},
									},
								},
							},
						},
					},
					TimeWindow: []v1.SLOTimeWindow{
						{
							Duration:  v1.NewDurationShorthand(1, v1.DurationShorthandUnitWeek),
							IsRolling: false,
							Calendar: &v1.SLOCalendar{
								StartTime: "2022-01-01 12:00:00",
								TimeZone:  "America/New_York",
							},
						},
					},
					BudgetingMethod: v1.SLOBudgetingMethodTimeslices,
					Objectives: []v1.SLOObjective{
						{
							DisplayName:     "Good",
							Target:          ptr(0.995),
							TimeSliceTarget: ptr(0.95),
							TimeSliceWindow: ptr(v1.NewDurationShorthand(1, v1.DurationShorthandUnitMinute)),
						},
					},
				},
			)},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			objects, err := Convert(tc.objects)
			if len(tc.errors) > 0 {
				require.Error(t, err)
				govytest.AssertError(t, err, tc.errors...)
			} else {
				govytest.AssertNoError(t, err)
				errs := manifest.Validate(objects)
				require.Empty(t, errs, "failed to validate Nobl9 objects")
			}
		})
	}
}

func listAllFilesInDir(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	return files
}

func ptr[T any](v T) *T { return &v }
