package clients

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"
	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
)

// Cache is an interface to a cache aws clients for all the
// roles specified by the exporter. For jobs with many duplicate roles, this provides
// relief to the AWS API and prevents timeouts by excessive credential requesting.
type Cache interface {
	GetCloudwatchClient(region string, role config.Role, concurrencyLimit int) cloudwatch_client.Client
	GetTaggingClient(region string, role config.Role, concurrencyLimit int) tagging.Client
	GetAccountClient(region string, role config.Role) account.Client

	Refresh()
	Clear()
}
