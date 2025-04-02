package openslotonobl9

import (
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func setDefaults(jsonObject string) (result string, err error) {
	if gjson.Get(jsonObject, "metadata.project").String() == "" {
		jsonObject, err = sjson.Set(jsonObject, "metadata.project", "default")
		if err != nil {
			return "", errors.Wrap(err, "failed to set metadata.project to default")
		}
	}
	return jsonObject, nil
}
