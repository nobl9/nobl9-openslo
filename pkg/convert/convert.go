package convert

import (
	"bytes"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/nobl9/nobl9-openslo/internal/jsonpath"
)

func OpenSLOToNobl9(opensloData []byte, format openslosdk.ObjectFormat) ([]byte, error) {
	opensloObjects, err := openslosdk.Decode(bytes.NewReader(opensloData), format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode OpenSLO objects in %s format", format)
	}
	if len(opensloObjects) == 0 {
		return nil, errors.New("no OpenSLO objects found")
	}

	var buf bytes.Buffer
	if err = openslosdk.Encode(&buf, openslosdk.FormatJSON, opensloObjects...); err != nil {
		return nil, errors.Wrap(err, "failed to encode OpenSLO objects to JSON")
	}

	r := gjson.Parse(buf.String())

	var objects []gjson.Result
	if r.IsArray() {
		objects = r.Array()
	} else {
		objects = []gjson.Result{r}
	}

	nobl9Objects := make([]string, 0, len(objects))
	for _, object := range objects {
		walker := jsonpath.NewWalker()
		walker.Walk(object, "")
		paths := walker.Paths()

		rules, err := getConversionRules(
			object.Get("apiVersion").String(),
			object.Get("kind").String(),
		)
		if err != nil {
			return nil, err
		}

		nobl9Object := "{}"
		for path, value := range paths {
			if !rules.HasRuleFor(path) {
				continue
			}
			nobl9Object, err = rules.Convert(nobl9Object, path, value)
			if err != nil {
				return nil, err
			}
		}
		if gjson.Get(nobl9Object, "metadata.project").String() == "" {
			nobl9Object, err = sjson.Set(nobl9Object, "metadata.project", "default")
			if err != nil {
				return nil, errors.Wrap(err, "failed to set metadata.project to default")
			}
		}
		nobl9Objects = append(nobl9Objects, nobl9Object)
	}
	return []byte("[" + strings.Join(nobl9Objects, ",") + "]"), nil
}
