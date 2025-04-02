package openslov1

import (
	"bytes"
	"cmp"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/nobl9/nobl9-openslo/internal/annotations"
	"github.com/nobl9/nobl9-openslo/internal/jsonpath"
)

func ToNobl9(opensloData []byte, format openslosdk.ObjectFormat) ([]byte, error) {
	opensloObjects, err := openslosdk.Decode(bytes.NewReader(opensloData), format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode OpenSLO objects in %s format", format)
	}
	if len(opensloObjects) == 0 {
		return nil, errors.New("no OpenSLO objects found")
	}

	var buf bytes.Buffer
	if err = openslosdk.Encode(&buf, openslosdk.FormatJSON, opensloObjects...); err != nil {
		return nil, errors.Wrap(err, "failed to encode OpenSLO objects to JSON")
	}

	r := gjson.Parse(buf.String())

	var objects []gjson.Result
	if r.IsArray() {
		objects = r.Array()
	} else {
		objects = []gjson.Result{r}
	}

	nobl9Objects := make([]string, 0, len(objects))
	for _, object := range objects {
		nobl9Object, err := opensloObjectToNobl9(object)
		if err != nil {
			return nil, err
		}
		nobl9Objects = append(nobl9Objects, nobl9Object)
	}

	var v any
	if err = json.Unmarshal([]byte("["+strings.Join(nobl9Objects, ",")+"]"), &v); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal Nobl9 objects")
	}
	return json.Marshal(v)
}

func opensloObjectToNobl9(object gjson.Result) (nobl9Object string, err error) {
	walker := jsonpath.NewWalker()
	walker.Walk(object, "")
	paths := sortPaths(walker.Paths())

	opensloVersion := object.Get("apiVersion").String()
	rules, err := getConversionRules(
		opensloVersion,
		object.Get("kind").String(),
	)
	if err != nil {
		return "", err
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
