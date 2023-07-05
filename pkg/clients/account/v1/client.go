package v1

import (
	"context"
	"errors"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

type client struct {
	logger    logging.Logger
	stsClient stsiface.STSAPI
}

func NewClient(logger logging.Logger, stsClient stsiface.STSAPI) account.Client {
	return &client{
		logger:    logger,
		stsClient: stsClient,
	}
}

func (c client) GetAccount(ctx context.Context) (string, error) {
	result, err := c.stsClient.GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	if result.Account == nil {
		return "", errors.New("aws sts GetCallerIdentityWithContext returned no account")
	}
	return *result.Account, nil
}
