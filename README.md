# nobl9-openslo

[OpenSLO](https://openslo.com) converter for Nobl9!

Convert OpenSLO schema to
[Nobl9 configuration](https://docs.nobl9.com/yaml-guide) with ease 🚀

> [!TIP]
> Starting with [sloctl](https://github.com/nobl9/sloctl)
> [version 0.12.0](https://github.com/nobl9/sloctl/releases/tag/v0.12.0)
> OpenSLO converter is available as a CLI through `sloctl convert openslo` command.

## Installation

To add the latest version to your Go module run:

```sh
go get github.com/nobl9/nobl9-go
```

## Usage

```go
package main

import (
	"bytes"
	"context"
	"log"

	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/nobl9/nobl9-openslo/pkg/openslotonobl9"
)

const opensloData = `
apiVersion: openslo/v1
kind: Service
metadata:
  annotations:
    nobl9.com/metadata.project: non-default
    my.domain/custom: foo
  name: example-service
spec:
  description: Example service description
`

func main() {
	// Read OpenSLO objects.
	objects, err := openslosdk.Decode(bytes.NewBufferString(opensloData), openslosdk.FormatYAML)
	if err != nil {
		log.Fatalf("failed to read OpenSLO objects: %v", err)
	}
	// Convert OpenSLO to Nobl9.
	nobl9Objects, err := openslotonobl9.Convert(objects)
	if err != nil {
		log.Fatalf("failed to convert OpenSLO to Nobl9: %v", err)
	}
	// Create Nobl9 SDK client.
	client, err := sdk.DefaultClient()
	if err != nil {
		log.Fatalf("failed to create Nobl9 SKD client: %v", err)
	}
	// Apply the objects.
	if err = client.Objects().V1().Apply(context.Background(), nobl9Objects); err != nil {
		log.Fatalf("failed to apply Nobl9 objects: %v", err)
	}
}
```

## How it works

1. Resolve object references, by either inlining or exporting dependent objects.
2. Validate decoded and resolved OpenSLO objects according to OpenSLO-defined rules.
3. Validate decoded and resolved OpenSLO objects according to Nobl9-defined,
   custom rules.
4. Convert OpenSLO objects to Nobl9 objects.

### Objects mapping

The following OpenSLO objects map to Nobl9 schema:

<!-- markdownlint-disable MD013 -->
| OpenSLO object             | Nobl9 object        | Supported  | Extra rules                                                      |
|----------------------------|---------------------|:----------:|------------------------------------------------------------------|
| v1.Service                 | v1alpha.Service     |     ✅     |                                                                  |
| v1.SLO                     | v1alpha.SLO         |     ✅     |                                                                  |
| v1.SLI                     | -                   |    ✖️       | If referenced by SLO it will be inlined.                         |
| v1.DataSource              | v1alpha.Agent       |     ✅     | By default Agent is created, use annotations to create a Direct. |
| v1.AlertPolicy             | v1alpha.AlertPolicy |     ✅     |                                                                  |
| v1.AlertCondition          | -                   |    ✖️       | If referenced by AlertPolicy it will be inlined.                 |
| v1.AlertNotificationTarget | v1.AlertMethod      |     ✅     |                                                                  |
<!-- markdownlint-enable MD013 -->

In addition, there are some special rules for generic fields in OpenSLO schema.

#### v1.SLI

`spec..metricSource.spec` field is directly converted to a matching Nobl9
metric spec based on `metricSource.type` field.

Example:

```yaml
# OpenSLO input:
metricSource:
  type: prometheus
  spec:
    promql: sum(http_request_duration_seconds_count{handler="/api/v1/slos"})
# Nobl9 output:
query:
  prometheus:
    promql: sum(http_request_duration_seconds_count{handler="/api/v1/slos"})
```

You can see that each of the fields in the `metricSource.spec` must be exactly
what Nobl9 `query.<metricSource.type>` defines.

#### v1.DataSource

Similar to [_v1.SLI_](#v1sli), `spec.type` field is used to determine the type
of Nobl9 data source details and `spec.connectionDetails` content must match
the Nobl9 definition for that data source type.

Example:

```yaml
# OpenSLO input:
type: appDynamics
connectionDetails:
  accountName: nobl9
  clientID: dev-agent@nobl9
  clientName: dev-agent
  clientSecret: secret
  url: https://example.com
# Nobl9 output:
appDynamics:
  accountName: nobl9
  clientID: dev-agent@nobl9
  clientName: dev-agent
  clientSecret: secret
  url: https://example.com
```

### Inlining and exporting rules

The list of objects passed to `Convert` method must contain all objects
that are referenced by the objects in the list.
For instance, if `v1.SLO` named _my-slo_ references `v1.SLI` named _my-sli_,
then the list must contain `v1.SLI` named _my-sli_.

#### v1.SLO

- `spec.indicatorRef` inlines `v1.SLI`.
- `spec.alertPolicies[*].alertPolicyRef` inlines `v1.AlertPolicy`.
  The inlining is done recursively, so that objects referenced by `v1.AlertPolicy`
  are inlined in the inlined `v1.AlertPolicy`.
- `spec.alertPolicies[*]` (inlined version) is exported.

#### v1.AlertPolicy

- `spec.conditions[*].conditionRef` inlines `v1.AlertCondition`.
- `spec.notificationTargets[*]` (inlined version) is exported.

### Modifying Nobl9 objects

Each field in a resulting Nobl9 object can be modified through the use of
`metadata.annotations` field in OpenSLO object.
In order to change a certain field in a resulting Nobl9 object,
the user must provide an annotation with the following format:

```text
nobl9.com/<field_path>: <value>
```

Example:

```yaml
# Input:
apiVersion: openslo/v1
kind: Service
metadata:
  annotations:
    nobl9.com/metadata.project: non-default
    my.domain/custom: foo
  name: example-service
spec:
  description: Example service description
# Output:
apiVersion: n9/v1alpha
kind: Service
metadata:
  annotations:
    my.domain/custom: foo
  project: non-default
  name: example-service
spec:
  description: Example service description
```

Common use cases:

- `nobl9.com/metadata.project` - sets the project for the object.
- `nobl9.com/kind` - sets the service kind for the object.
  This is a special case, only allowed for `DataSource`, this way
  users can decide whether to convert `DataSource` to `Agent` or `Direct`.
