package account

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

type Client interface {
	GetAccount(ctx context.Context) (string, error)
}

type client struct {
	logger    logging.Logger
	stsClient stsiface.STSAPI
}

func NewClient(logger logging.Logger, stsClient stsiface.STSAPI) Client {
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
