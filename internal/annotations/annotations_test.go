package annotations

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func TestAddOpenSLOAnnotations(t *testing.T) {
	type FooStruct struct {
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}

	tests := []struct {
		name       string
		jsonObject string
		path       string
		value      any
		expected   string
	}{
		{
			name:       "string value",
			jsonObject: `{"metadata":{"annotations":{}}}`,
			path:       "path.to.annotation",
			value:      "value",
			expected:   `{"metadata":{"annotations":{"openslo.com/path.to.annotation":"value"}}}`,
		},
		{
			name:       "numeric value",
			jsonObject: `{}`,
			path:       "another.annotation",
			value:      123,
			expected:   `{"metadata":{"annotations":{"openslo.com/another.annotation":123}}}`,
		},
		{
			name:       "struct value",
			jsonObject: `{"metadata":{"annotations":{}}}`,
			path:       "struct.annotation",
			value:      FooStruct{"hello", 42},
			expected:   `{"metadata":{"annotations":{"openslo.com/struct.annotation":"{\"foo\":\"hello\",\"bar\":42}"}}}`,
		},
		{
			name:       "slice value",
			jsonObject: `{"metadata":{"annotations":{}}}`,
			path:       "slice.annotation",
			value:      []string{"a", "b", "c"},
			expected:   `{"metadata":{"annotations":{"openslo.com/slice.annotation":"[\"a\",\"b\",\"c\"]"}}}`,
		},
		{
			name:       "map value",
			jsonObject: `{"metadata":{"annotations":{}}}`,
			path:       "map.annotation",
			value:      map[string]any{"key": "val", "num": 100},
			expected:   `{"metadata":{"annotations":{"openslo.com/map.annotation":"{\"key\":\"val\",\"num\":100}"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AddOpenSLOToNobl9(tt.jsonObject, tt.path, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestThis(t *testing.T) {
	object := `{"A":{"B":[{"name":[{"C":"D"}]}, {"name":[{"C":"D"}]}]}}`
	result := gjson.Get(object, "A.B.#.name.#.C")
	var err error
	for i := range result.Int() {
		object, err = sjson.Set(object, fmt.Sprintf("A.B.%d.age", i), "baz")
		require.NoError(t, err)
	}
	fmt.Println(object)
}
