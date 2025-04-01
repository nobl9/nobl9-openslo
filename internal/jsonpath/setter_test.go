package jsonpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	tests := map[string]struct {
		object   string
		path     string
		value    any
		expected string
	}{
		"no hash in path": {
			object:   `{"A":{"B":"foo"}}`,
			path:     "A.B",
			value:    "bar",
			expected: `{"A":{"B":"bar"}}`,
		},
		"no hash in path, new child": {
			object:   `{"A":{"B":{}}}`,
			path:     "A.B.C",
			value:    "value",
			expected: `{"A":{"B":{"C":"value"}}}`,
		},
		"simple list": {
			object:   `{"A":["a","b"]}`,
			path:     "A.#",
			value:    "c",
			expected: `{"A":["c","c"]}`,
		},
		"list of objects": {
			object:   `{"A":[{"B":"C"},{"D":"E"}]}`,
			path:     "A.#.D",
			value:    "X",
			expected: `{"A":[{"B":"C","D":"X"},{"D":"X"}]}`,
		},
		"example from Go doc": {
			object:   `{"A":[{"B":[{"C":"D"},{"C":"D"}]},{"B":[{"C":"D"},{"C":"D"}]}]}`,
			path:     "A.#.B.#.E",
			value:    "X",
			expected: `{"A":[{"B":[{"C":"D","E":"X"},{"C":"D","E":"X"}]},{"B":[{"C":"D","E":"X"},{"C":"D","E":"X"}]}]}`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := Set(tc.object, tc.path, tc.value)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
