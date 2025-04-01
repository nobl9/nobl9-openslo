package conversionrules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		yamlPath    string
		concrete    string
		shouldMatch bool
	}{
		{
			yamlPath:    "",
			concrete:    "",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.c",
			concrete:    "a.b.c",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.c",
			concrete:    "a.b.d",
			shouldMatch: false,
		},
		{
			yamlPath:    "a.b.#",
			concrete:    "a.b.c",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.#",
			concrete:    "a.b.1",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.#.#",
			concrete:    "a.b.x.10",
			shouldMatch: true,
		},
		{
			yamlPath:    "a.b.c.#.d.#",
			concrete:    "a.b.c.10.d.#",
			shouldMatch: true,
		},
	}
	for _, test := range tests {
		t.Run(test.yamlPath+"="+test.concrete, func(t *testing.T) {
			m := matchPath(test.yamlPath, test.concrete)
			assert.Equal(t, test.shouldMatch, m)
		})
	}
}
