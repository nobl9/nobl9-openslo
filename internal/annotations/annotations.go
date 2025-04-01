package annotations

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

// AddOpenSLOToNobl9 adds OpenSLO annotations to the given Nobl9 JSON object.
// Annotations are added to metadata.annotations.<key>, where key is of the following format:
//
//	openslo.com/<path>
//
// Example:
//
//	AddOpenSLOToNobl9(jsonObject, "path.to.annotation", "value") ->
//	`{"metadata":{"annotations":{"openslo.com/path.to.annotation":"value"}}}`
//
// Complex objects are marshaled to JSON before being added to the object.
func AddOpenSLOToNobl9(jsonObject, path string, value any) (string, error) {
	rt := reflect.TypeOf(value)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	var (
		data any
		err  error
	)
	switch rt.Kind() {
	case reflect.Slice, reflect.Map, reflect.Struct, reflect.Array:
		data, err = json.Marshal(value)
		if err != nil {
			return "", errors.Wrapf(err, "failed to marshal value for path %s", path)
		}
	default:
		data = value
	}
	annotationKey := escapeDots("openslo.com/" + path)
	return sjson.Set(jsonObject, "metadata.annotations."+annotationKey, data)
}

func escapeDots(path string) string {
	return strings.ReplaceAll(path, ".", "\\.")
}
