package openslotonobl9

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Convert(tc.objects)
			require.Error(t, err)
			govytest.AssertError(t, err, tc.errors...)
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
