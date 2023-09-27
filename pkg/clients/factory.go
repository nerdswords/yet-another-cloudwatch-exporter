package clients

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"
	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
)

// Factory is an interface to abstract away all logic required to produce the different
// YACE specific clients which wrap AWS clients
type Factory interface {
	GetCloudwatchClient(region string, role config.Role, concurrency cloudwatch_client.ConcurrencyConfig) cloudwatch_client.Client
	GetTaggingClient(region string, role config.Role, concurrencyLimit int) tagging.Client
	GetAccountClient(region string, role config.Role) account.Client
}
