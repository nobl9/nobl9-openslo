package conversionrules

import (
	"github.com/tidwall/sjson"
)

type Converter interface {
	Convert(jsonObject, path string, v any) (updatedJSON string, err error)
}

func Noop() Converter {
	return noopConverter{}
}

type noopConverter struct{}

func (c noopConverter) Convert(jsonObject, path string, v any) (updatedJSON string, err error) {
	return sjson.Set(jsonObject, path, v)
}

func Path(path string) Converter {
	return pathConverter{path: path}
}

type pathConverter struct {
	path string
}

func (c pathConverter) Convert(jsonObject, _ string, v any) (updatedJSON string, err error) {
	return sjson.Set(jsonObject, c.path, v)
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
	return sjson.Set(jsonObject, path, convertedValue)
}

func Full(f func(jsonObject, path string, v any) (updatedJSON string, err error)) Converter {
	return fullConverter{f: f}
}

type fullConverter struct {
	f func(jsonObject, path string, v any) (updatedJSON string, err error)
}

func (c fullConverter) Convert(jsonObject, path string, v any) (updatedJSON string, err error) {
	return c.f(jsonObject, path, v)
}
