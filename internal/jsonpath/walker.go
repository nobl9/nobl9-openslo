package jsonpath

import "github.com/tidwall/gjson"

func NewWalker() *Walker {
	return &Walker{
		paths: make(map[string]any),
	}
}

type Walker struct {
	paths map[string]any
}

func (w *Walker) Paths() map[string]any {
	return w.paths
}

func (w *Walker) Walk(r gjson.Result, path string) {
	r.ForEach(func(key, value gjson.Result) bool {
		var pathElement string
		pathElement = key.String()

		var currentPath string
		switch {
		case path != "":
			currentPath = path + "." + pathElement
		default:
			currentPath = pathElement
		}
		w.paths[currentPath] = value.Value()

		if !value.IsArray() && !value.IsObject() {
			return true
		}
		w.Walk(value, currentPath)
		return true
	})
}
