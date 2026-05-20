package openslotonobl9

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func setDefaults(jsonObject string) (result string, err error) {
	if gjson.Get(jsonObject, "metadata.project").String() == "" {
		jsonObject, err = sjson.Set(jsonObject, "metadata.project", "default")
		if err != nil {
			return "", fmt.Errorf("failed to set metadata.project to default: %w", err)
		}
	}
	return jsonObject, nil
}
