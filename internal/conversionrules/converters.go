package conversionrules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nobl9/nobl9-openslo/internal/jsonpath"
)

type ConversionFunc func(jsonObject, path string, v any) (updatedJSON string, err error)

type Converter interface {
	Convert(jsonObject, path string, v any) (updatedJSON string, err error)
}

func Noop() Converter {
	return noopConverter{}
}

type noopConverter struct{}

func (c noopConverter) Convert(jsonObject, _ string, _ any) (updatedJSON string, err error) {
	return jsonObject, nil
}

func Direct() Converter {
	return directConverter{}
}

type directConverter struct{}

func (c directConverter) Convert(jsonObject, path string, v any) (updatedJSON string, err error) {
	return jsonpath.Set(jsonObject, path, v)
}

func Path(path string) Converter {
	return pathConverter{path: path}
}

type pathConverter struct {
	path string
}

func (c pathConverter) Convert(jsonObject, _ string, v any) (updatedJSON string, err error) {
	return jsonpath.Set(jsonObject, c.path, v)
}

func PathIndex(format string) Converter {
	return pathIndexConverter{format: format}
}

type pathIndexConverter struct {
	format string
}

func (c pathIndexConverter) Convert(jsonObject, path string, v any) (updatedJSON string, err error) {
	split := strings.Split(path, pathSeparator)
	var indices []any
	for _, s := range split {
		if i, err := strconv.Atoi(s); err == nil {
			indices = append(indices, i)
		}
	}
	newPath := fmt.Sprintf(c.format, indices...)
	if strings.Contains(newPath, "(MISSING)") {
		return "", fmt.Errorf("path %q is missing index (format: %q)", path, c.format)
	}
	return jsonpath.Set(jsonObject, newPath, v)
}

func Value(f func(v any) (any, error)) Converter {
	return valueConverter{f: f}
}

type valueConverter struct {
	f func(v any) (any, error)
}

func (c valueConverter) Convert(jsonObject, path string, v any) (updatedJSON string, err error) {
	convertedValue, err := c.f(v)
	if err != nil {
		return "", err
	}
	return jsonpath.Set(jsonObject, path, convertedValue)
}

func Custom(f ConversionFunc) Converter {
	return customConverter{f: f}
}

type customConverter struct {
	f ConversionFunc
}

func (c customConverter) Convert(jsonObject, path string, v any) (updatedJSON string, err error) {
	return c.f(jsonObject, path, v)
}
