package jsonpath

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Set sets the value at the provided path in the object.
// Unlike [sjson.Set] it supports setting values in arrays using the hash character `#`.
//
// Given:
//
//	path   = A.#.B.#.E
//	object = {"A":[{"B":[{"C":"D"},{"C","D"}]},{"B":[{"C":"D"},{"C","D"}]}]
//	value  = "X"
//
// It will generate:
//
//	{
//	  "A": [
//	    {
//	      "B": [
//	        {"C": "D", "E": "X"},
//	        {"C": "D", "E": "X"}
//	      ]
//	    },
//	    {
//	      "B": [
//	        {"C": "D", "E": "X"},
//	        {"C": "D", "E": "X"}
//	      ]
//	    }
//	  ]
//	}
//
// This means every leaf node in the path will be set to the provided value.
func Set(object, path string, value any) (string, error) {
	hashIdx := strings.Index(path, "#")
	if hashIdx == -1 {
		return sjson.Set(object, path, value)
	}
	result := gjson.Get(object, path[:hashIdx+1])
	var err error
	for i := range int(result.Int()) {
		object, err = Set(object, strings.Replace(path, "#", strconv.Itoa(i), 1), value)
		if err != nil {
			return "", err
		}
	}
	return object, nil
}
