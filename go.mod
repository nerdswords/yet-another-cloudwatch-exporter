module github.com/nerdswords/yet-another-cloudwatch-exporter

go 1.22.0

require (
	github.com/aws/aws-sdk-go v1.50.26
	github.com/aws/aws-sdk-go-v2 v1.25.2
	github.com/aws/aws-sdk-go-v2/config v1.27.4
	github.com/aws/aws-sdk-go-v2/credentials v1.17.4
	github.com/aws/aws-sdk-go-v2/service/amp v1.25.1
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.40.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.36.1
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.38.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.149.1
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.21.1
	github.com/aws/aws-sdk-go-v2/service/shield v1.25.1
	github.com/aws/aws-sdk-go-v2/service/storagegateway v1.27.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.1
	github.com/aws/smithy-go v1.20.1
	github.com/go-kit/log v0.2.1
	github.com/grafana/regexp v0.0.0-20221123153739-15dc172cd2db
	github.com/prometheus/client_golang v1.18.0
	github.com/prometheus/common v0.47.0
	github.com/stretchr/testify v1.8.4
	github.com/urfave/cli/v2 v2.27.1
	golang.org/x/sync v0.6.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.23.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/sys v0.16.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
