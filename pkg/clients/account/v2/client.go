package v2

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

type client struct {
	logger    logging.Logger
	stsClient *sts.Client
}

func NewClient(logger logging.Logger, stsClient *sts.Client) account.Client {
	return &client{
		logger:    logger,
		stsClient: stsClient,
	}
}

func (c client) GetAccount(ctx context.Context) (string, error) {
	result, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	if result.Account == nil {
		return "", errors.New("aws sts GetCallerIdentityWithContext returned no account")
	}
	return *result.Account, nil
}
