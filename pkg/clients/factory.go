package clients

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"
	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// Factory is an interface to abstract away all logic required to produce the different
// YACE specific clients which wrap AWS clients
type Factory interface {
	GetCloudwatchClient(region string, role model.Role, concurrency cloudwatch_client.ConcurrencyConfig) cloudwatch_client.Client
	GetTaggingClient(region string, role model.Role, concurrencyLimit int) tagging.Client
	GetAccountClient(region string, role model.Role) account.Client
}
