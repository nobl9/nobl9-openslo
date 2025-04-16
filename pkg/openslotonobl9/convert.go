package openslotonobl9

import (
	"bytes"
	"cmp"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/nobl9/nobl9-openslo/internal/annotations"
	"github.com/nobl9/nobl9-openslo/internal/jsonpath"
)

func Convert(opensloData []byte) ([]manifest.Object, error) {
	objects, err := openslosdk.Decode(bytes.NewReader(opensloData), openslosdk.FormatYAML)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode OpenSLO objects in %s format", openslosdk.FormatYAML)
	}
	if len(objects) == 0 {
		return nil, errors.New("no OpenSLO objects found")
	}
	objects, err = resolveObjectReferences(objects)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve OpenSLO object references")
	}
	if err = openslosdk.Validate(objects...); err != nil {
		return nil, errors.Wrapf(err, "failed to validate OpenSLO objects")
	}

	nobl9JSONObjects := make([]string, 0, len(objects))
	for _, object := range objects {
		jsonObject, err := opensloObjectToNobl9(object)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert object from OpenSLO to Nobl9 format")
		}
		nobl9JSONObjects = append(nobl9JSONObjects, jsonObject)
	}
	return sdk.DecodeObjects([]byte("[" + strings.Join(nobl9JSONObjects, ",") + "]"))
}

func opensloObjectToNobl9(opensloObject openslo.Object) (nobl9Object string, err error) {
	var buf bytes.Buffer
	if err = openslosdk.Encode(&buf, openslosdk.FormatJSON, opensloObject); err != nil {
		return "", errors.Wrap(err, "failed to encode OpenSLO objects to JSON")
	}
	object := gjson.Parse(buf.String()).Array()[0]

	walker := jsonpath.NewWalker()
	walker.Walk(object, "")
	paths := sortPaths(walker.Paths())

	opensloVersion, err := openslo.ParseVersion(object.Get("apiVersion").String())
	if err != nil {
		return "", err
	}
	opensloKind, err := openslo.ParseKind(object.Get("kind").String())
	if err != nil {
		return "", err
	}
	rules, err := getConversionRules(opensloVersion, opensloKind)
	if err != nil {
		return "", err
	}
	if len(rules) == 0 {
		fmt.Fprintf(os.Stderr, "no conversion rules for %s %s, skipping\n", opensloVersion, opensloKind)
		return "", nil
	}

	if err = validateOpenSLOObject(object, opensloVersion, opensloKind); err != nil {
		return "", errors.Wrapf(err, "failed to validate OpenSLO object")
	}

	nobl9Object = "{}"
	for _, path := range paths {
		nobl9Object, err = rules.Convert(nobl9Object, path.Path, path.Value)
		if err != nil {
			return "", err
		}
	}
	nobl9Object, err = setDefaults(nobl9Object)
	if err != nil {
		return "", err
	}
	return annotations.AddOpenSLOToNobl9(nobl9Object, "apiVersion", opensloVersion)
}

type pathTuple struct {
	Path  string
	Value any
}

func sortPaths(pathsMap map[string]any) []pathTuple {
	// The first item from this list will be the last in the result.
	reversePrecedence := map[string]func(s1, s2 string) bool{
		"metadata.annotations":                strings.HasPrefix,
		"spec.indicator.spec.ratioMetric":     strings.HasPrefix,
		"spec.indicator.spec.thresholdMetric": strings.HasPrefix,
	}

	keys := slices.SortedFunc(maps.Keys(pathsMap), func(s1 string, s2 string) int {
		for p, cmpFunc := range reversePrecedence {
			cmp1, cmp2 := cmpFunc(s1, p), cmpFunc(s2, p)
			if cmp1 && !cmp2 {
				return 1
			}
			if !cmp1 && cmp2 {
				return -1
			}
		}
		return cmp.Compare(s1, s2)
	})

	result := make([]pathTuple, 0, len(keys))
	for _, key := range keys {
		result = append(result, pathTuple{
			Path:  key,
			Value: pathsMap[key],
		})
	}
	return result
}
