package openslotonobl9

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/goccy/go-yaml"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	inputsDir  = "./test_data/inputs/"
	outputsDir = "./test_data/outputs/"
)

func TestToNobl9(t *testing.T) {
	inputs := listAllFilesInDir(t, inputsDir)
	outputs := listAllFilesInDir(t, outputsDir)
	require.Len(t, inputs, len(outputs))

	for _, fileName := range inputs {
		t.Run(fileName, func(t *testing.T) {
			inputFileData, err := os.ReadFile(filepath.Join(inputsDir, fileName))
			require.NoError(t, err)

			outputsFileData, err := os.ReadFile(filepath.Join(outputsDir, fileName))
			require.NoError(t, err)

			actual, err := Convert(inputFileData, openslosdk.FormatYAML)
			require.NoError(t, err)

			expectedJSON, err := yaml.YAMLToJSON(outputsFileData)
			require.NoError(t, err)
			assert.JSONEq(t, string(expectedJSON), string(actual))

			opensloObjects, err := openslosdk.Decode(bytes.NewReader(inputFileData), openslosdk.FormatYAML)
			require.NoError(t, err)
			err = openslosdk.Validate(opensloObjects...)
			require.NoError(t, err, "failed to validate OpenSLO objects")

			nobl9Objects, err := sdk.DecodeObjects(actual)
			require.NoError(t, err)
			errs := manifest.Validate(nobl9Objects)
			require.Empty(t, errs, "failed to validate Nobl9 objects")
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
